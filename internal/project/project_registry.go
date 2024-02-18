package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/go-git/go-billy/v5/util"
	"github.com/inoxlang/inox/internal/buntdb"
	"github.com/inoxlang/inox/internal/core"
	"github.com/inoxlang/inox/internal/globals/fs_ns"
	"github.com/inoxlang/inox/internal/inoxd/node"
	"github.com/inoxlang/inox/internal/project/cloudflareprovider"
	"github.com/inoxlang/inox/internal/project/scaffolding"
)

const (
	KV_FILENAME = "projects.kv"

	DEV_OS_DIR           = "dev"
	DEV_DATABASES_OS_DIR = "databases"
)

var (
	ErrProjectNotFound                = errors.New("project not found")
	ErrProjectPersistenceNotAvailable = errors.New("project persistence is not available")
)

type Registry struct {
	projectsDir string //projects directory on the OS filesystem
	filesystem  *fs_ns.OsFilesystem
	metadata    *buntdb.DB

	openProjects        map[core.ProjectID]*Project
	openProjectsLock    sync.Mutex
	openProjectsContext *core.Context

	//TODO: close idle projects (no FS access AND no provider-related accesses AND no production program running)
}

// OpenRegistry opens a project registry located on the OS filesystem.
func OpenRegistry(projectsDir string, openProjectsContext *core.Context) (*Registry, error) {
	kvPath := filepath.Join(projectsDir, KV_FILENAME)

	kv, err := buntdb.OpenBuntDBNoPermCheck(kvPath, fs_ns.GetOsFilesystem())

	if err != nil {
		return nil, fmt.Errorf("failed to open database of projects: %w", err)
	}

	return &Registry{
		projectsDir:         projectsDir,
		metadata:            kv,
		filesystem:          fs_ns.GetOsFilesystem(),
		openProjects:        map[core.ProjectID]*Project{},
		openProjectsContext: openProjectsContext,
	}, nil
}

func (r *Registry) Close(ctx *core.Context) {
	r.metadata.Close()
}

type CreateProjectParams struct {
	Name       string `json:"name"`
	Template   string `json:"template,omitempty"`
	AddTutFile bool   `json:"addTutFile,omitempty"`
}

// CreateProject
func (r *Registry) CreateProject(ctx *core.Context, params CreateProjectParams) (core.ProjectID, error) {
	if matched, err := regexp.MatchString(PROJECT_NAME_REGEX, params.Name); !matched || err != nil {
		return "", ErrInvalidProjectName
	}
	id := core.RandomProjectID(params.Name)

	// create the directory for storing projects if necessary
	err := r.filesystem.MkdirAll(r.projectsDir, fs_ns.DEFAULT_DIR_FMODE)
	if err != nil {
		return "", fmt.Errorf("failed to create directory to store projects: %w", err)
	}

	// persist data

	r.persistProjectData(ctx, id, projectData{CreationParams: params})

	// create the project's directory
	projectDir := r.filesystem.Join(r.projectsDir, string(id))
	err = r.filesystem.MkdirAll(projectDir, fs_ns.DEFAULT_DIR_FMODE)

	if err != nil {
		return "", fmt.Errorf("failed to create directory for project %s: %w", id, err)
	}

	// create initial files
	projectFS, err := fs_ns.OpenMetaFilesystem(ctx, r.filesystem, fs_ns.MetaFilesystemParams{
		Dir: projectDir,
	})
	if err != nil {
		return "", fmt.Errorf("failed to open the project filesystem to write initial files %s: %w", id, err)
	}

	defer projectFS.Close(ctx)

	if params.Template != "" {
		if err := scaffolding.WriteTemplate(params.Template, projectFS); err != nil {
			return "", fmt.Errorf("failed to write template %q to the project filesystem: %w", params.Template, err)
		}
	}

	if params.AddTutFile {
		util.WriteFile(projectFS, DEFAULT_TUT_FILENAME, []byte(nil), fs_ns.DEFAULT_DIR_FMODE)
	}

	//create a directory for storing the project's dev databases

	_, err = r.getCreateDevDatabasesDir(id)
	if err != nil {
		return "", err
	}

	return id, nil
}

func (r *Registry) getCreateDevDatabasesDir(id core.ProjectID) (projectDevDatabasesDir string, err error) {
	//create the dev dir that will store the dev databases

	devDir := filepath.Join(r.projectsDir, DEV_OS_DIR)
	err = r.filesystem.MkdirAll(devDir, fs_ns.DEFAULT_DIR_FMODE)
	if err != nil {
		return
	}

	//create the <dev dir>/<project id>/databases dir
	projectDevDatabasesDir = filepath.Join(devDir, string(id), DEV_DATABASES_OS_DIR)
	err = r.filesystem.MkdirAll(projectDevDatabasesDir, fs_ns.DEFAULT_DIR_FMODE)
	if err != nil {
		projectDevDatabasesDir = ""
		return
	}

	return projectDevDatabasesDir, nil
}

type OpenProjectParams struct {
	Id                core.ProjectID
	DevSideConfig     DevSideProjectConfig `json:"config"`
	TempTokens        *TempProjectTokens   `json:"tempTokens,omitempty"`
	MaxFilesystemSize core.ByteCount       `json:"-"`
	ExposeWebServers  bool
}

func (r *Registry) OpenProject(ctx *core.Context, params OpenProjectParams) (*Project, error) {
	r.openProjectsLock.Lock()
	defer r.openProjectsLock.Unlock()

	openProjects := r.openProjects

	if project, ok := openProjects[params.Id]; ok {
		return project, nil
	}

	var serializedProjectData string
	var found bool

	err := r.metadata.View(func(tx *buntdb.Tx) error {
		projectKey := getProjectKvKey(params.Id)
		data, err := tx.Get(string(projectKey))
		if errors.Is(err, buntdb.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		serializedProjectData = data
		found = true
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error while reading KV: %w", err)
	}

	if !found {
		return nil, ErrProjectNotFound
	}

	config := ProjectConfiguration{
		ExposeWebServers: params.ExposeWebServers,
	}

	// get project data from the database

	var projectData projectData
	err = json.Unmarshal([]byte(serializedProjectData), &projectData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal project's data: %w", err)
	}

	if projectData.Applications == nil {
		projectData.Applications = map[node.ApplicationName]*applicationData{}
	}

	if projectData.Secrets == nil {
		projectData.Secrets = map[core.SecretName]localSecret{}
	}

	// open the project's filesystem

	projectDir := r.filesystem.Join(r.projectsDir, string(params.Id))
	projectFS, err := fs_ns.OpenMetaFilesystem(r.openProjectsContext, r.filesystem, fs_ns.MetaFilesystemParams{
		Dir:            projectDir,
		MaxUsableSpace: params.MaxFilesystemSize,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to open filesystem of project %s: %w", params.Id, err)
	}

	project := &Project{
		id:             params.Id,
		liveFilesystem: projectFS,
		tempTokens:     params.TempTokens,
		data:           projectData,
		persistFn:      r.persistProjectData,

		storeSecretsInProjectData: true,

		config: config,
	}

	if params.DevSideConfig.Cloudflare != nil {
		cf, err := cloudflareprovider.New(project.id, params.DevSideConfig.Cloudflare)
		if err != nil {
			return nil, fmt.Errorf("failed to create clouflare helper: %w", err)
		}
		project.cloudflare = cf
	}

	project.Share(nil)
	r.openProjects[project.id] = project

	projectDevDatabasesDir, err := r.getCreateDevDatabasesDir(project.id)
	if err != nil {
		return nil, err
	}

	project.devDatabasesDirOnOsFs.Store(projectDevDatabasesDir)

	return project, nil
}

func (r *Registry) persistProjectData(ctx *core.Context, id core.ProjectID, data projectData) error {
	serialized, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal project data: %w", err)
	}

	return r.metadata.Update(func(tx *buntdb.Tx) error {
		key := getProjectKvKey(id)
		_, _, err := tx.Set(string(key), string(serialized), nil)
		return err
	})
}
