package internal

import (
	"errors"
	"fmt"
	"strings"

	parse "github.com/inox-project/inox/internal/parse"
)

type Bytecode struct {
	NotClonableMixin

	module    *Module
	constants []Value
	main      *CompiledFunction
}

func (b *Bytecode) Constants() []Value {
	return b.constants
}

func (b *Bytecode) FormatInstructions(ctx *Context, leftPadding string) []string {
	return FormatInstructions(ctx, b.main.Instructions, 0, leftPadding, b.constants)
}

// FormatConstants returns a human readable representation of compiled constants.
func (b *Bytecode) FormatConstants(ctx *Context, leftPadding string) (output []string) {

	for cidx, cn := range b.constants {
		switch cn := cn.(type) {
		case *InoxFunction:
			output = append(output, fmt.Sprintf("%s[% 3d] (Compiled Function|%p)", leftPadding, cidx, &cn))
			for _, l := range FormatInstructions(ctx, cn.compiledFunction.Instructions, 0, leftPadding, nil) {
				output = append(output, fmt.Sprintf("%s     %s", leftPadding, l))
			}
		case *Bytecode:
			output = append(output, fmt.Sprintf("     %s", cn.Format(ctx, leftPadding+"    ")))
		default:
			repr := ""
			if cn.HasRepresentation(map[uintptr]int{}, &ReprConfig{}) {
				repr = string(GetRepresentation(cn, ctx))
			} else {
				repr = Stringify(cn, nil)
			}
			output = append(output, fmt.Sprintf("%s[% 3d] %s", leftPadding, cidx, repr))
		}
	}
	return
}

// Fomat returnsa a human readable representations of the bytecode.
func (b *Bytecode) Format(ctx *Context, leftPadding string) string {
	s := fmt.Sprintf("compiled constants:\n%s", strings.Join(b.FormatConstants(ctx, leftPadding), "\n"))
	s += fmt.Sprintf("\ncompiled instructions:\n%s\n", strings.Join(b.FormatInstructions(ctx, leftPadding), "\n"))
	return s
}

type CompiledFunction struct {
	ParamCount   int
	IsVariadic   bool
	LocalCount   int // includes parameters
	Instructions []byte
	SourceMap    map[int]instructionSourcePosition
	Bytecode     *Bytecode //bytecode containing the function
}

func (fn *CompiledFunction) GetSourcePosition(ip int) parse.SourcePosition {
	info := fn.SourceMap[ip]
	if info.chunk == nil {
		return parse.SourcePosition{
			SourceName: "??",
			Line:       1,
			Column:     1,
			Span:       parse.NodeSpan{Start: 0, End: 1},
		}
	}
	return info.chunk.GetSourcePosition(info.span)
}

type instructionSourcePosition struct {
	chunk *parse.ParsedChunk
	span  parse.NodeSpan
}

type InstructionCallbackFn = func(instr []byte, op Opcode, operands []int, constantIndexOperandIndex []int, constants []Value, i int) ([]byte, error)

// MapInstructions iterates instructions and calls callbackFn for each instruction.
func MapInstructions(b []byte, constants []Value, callbackFn InstructionCallbackFn) ([]byte, error) {
	i := 0

	var newInstructions []byte

	for i < len(b) {
		op := Opcode(b[i])
		numOperands := OpcodeOperands[b[i]]
		operands, read := ReadOperands(numOperands, b[i+1:])

		var referredConstants []Value
		var constantIndexOperandIndexes []int

		if len(constants) != 0 {
			for j, operand := range operands {
				if OpcodeConstantIndexes[b[i]][j] {
					if numOperands[j] != 2 {
						return nil, errors.New("index of constant should have a width of 2, opcode: " + OpcodeNames[op])
					}
					referredConstants = append(referredConstants, constants[operand])
					constantIndexOperandIndexes = append(constantIndexOperandIndexes, j)
				}
			}
		}

		instruction := b[i : i+1+read]
		if instr, err := callbackFn(instruction, op, operands, constantIndexOperandIndexes, referredConstants, i); err != nil {
			return nil, err
		} else {
			newInstructions = append(newInstructions, instr...)
		}

		i += 1 + read
	}

	return newInstructions, nil
}

// Opcode represents a single byte operation code.
type Opcode = byte

// opcodes
const (
	OpPushConstant Opcode = iota
	OpPop
	OpCopyTop
	OpSwap
	OpPushTrue
	OpPushFalse
	OpEqual
	OpNotEqual
	OpIs
	OpIsNot
	OpMinus
	OpBooleanNot
	OpMatch
	OpGroupMatch
	OpIn
	OpSubstrOf
	OpKeyOf
	OpDoSetDifference
	OpJumpIfFalse
	OpAndJump // Logical AND jump
	OpOrJump  // Logical OR jump
	OpJump
	OpPushNil
	OpCreateList
	OpCreateKeyList
	OpCreateTuple
	OpCreateObject
	OpCreateRecord
	OpCreateDict
	OpCreateMapping
	OpCreateUData
	OpCreateUdataHiearchyEntry
	OpSpreadObject
	OpExtractProps
	OpSpreadList
	OpSpreadTuple
	OpAppend
	OpCreateListPattern
	OpCreateObjectPattern
	OpCreateOptionPattern
	OpCreateUnionPattern
	OpCreateStringUnionPattern
	OpCreateRepeatedPatternElement
	OpCreateSequenceStringPattern
	OpCreatePatternNamespace
	OpCreateOptionalPattern
	OpToPattern
	OpToBool
	OpCreateCheckedString
	OpCreateOption
	OpCreatePath
	OpCreatePathPattern
	OpCreateURL
	OpCreateHost
	OpCreateRuneRange
	OpCreateIntRange
	OpCreateUpperBoundRange
	OpCreateTestSuite
	OpCreateTestCase
	OpCreateLifetimeJob
	OpCreateReceptionHandler
	OpSendValue
	OpSpreadObjectPattern
	BindCapturedLocals
	OpCall
	OpReturn
	OpYield
	OpCallPattern
	OpDropPerms
	OpSpawnRoutine
	OpImport
	OpGetGlobal
	OpSetGlobal
	OpGetLocal
	OpSetLocal
	OpGetSelf
	OpGetSupersys
	OpResolveHost
	OpAddHostAlias
	OpResolvePattern
	OpAddPattern
	OpResolvePatternNamespace
	OpAddPatternNamespace
	OpPatternNamespaceMemb
	OpSetMember
	OpSetIndex
	OpSetSlice
	OpIterInit
	OpIterNext
	OpIterNextChunk
	OpIterKey
	OpIterValue
	OpIterPrune
	OpWalkerInit
	OpIntBin
	OpFloatBin
	OpNumBin
	OptStrConcat
	OpConcat
	OpRange
	OpMemb
	OpDynMemb
	OpAt
	OpSlice
	OpAssert
	OpBlockLock
	OpBlockUnlock
	OpSuspendVM
)

// OpcodeNames are string representation of opcodes.
var OpcodeNames = [...]string{
	OpPushConstant:                 "PUSH_CONST",
	OpPop:                          "POP",
	OpCopyTop:                      "COPY_TOP",
	OpSwap:                         "SWAP",
	OpPushTrue:                     "PUSH_TRUE",
	OpPushFalse:                    "PUSH_FALSE",
	OpEqual:                        "EQUAL",
	OpNotEqual:                     "NOT_EQUAL",
	OpIs:                           "IS",
	OpIsNot:                        "IS_NOT",
	OpMinus:                        "NEG",
	OpBooleanNot:                   "NOT",
	OpMatch:                        "MATCH",
	OpGroupMatch:                   "GRP_MATCH",
	OpIn:                           "IN",
	OpSubstrOf:                     "SUBSTR_OF",
	OpKeyOf:                        "KEY_OF",
	OpDoSetDifference:              "DO_SET_DIFF",
	OpJumpIfFalse:                  "JUMP_IFF",
	OpAndJump:                      "AND_JUMP",
	OpOrJump:                       "OR_JUMP",
	OpJump:                         "JUMP",
	OpPushNil:                      "PUSH_NIL",
	OpCreateList:                   "CRT_LST",
	OpCreateKeyList:                "CRT_KLST",
	OpCreateTuple:                  "CRT_TUPLE",
	OpCreateObject:                 "CRT_OBJ",
	OpCreateRecord:                 "CRT_REC",
	OpCreateDict:                   "CRT_DICT",
	OpCreateMapping:                "CRT_MPG",
	OpCreateUData:                  "CRT_UDAT",
	OpCreateUdataHiearchyEntry:     "CRT_UDHE",
	OpSpreadObject:                 "SPREAD_OBJ",
	OpExtractProps:                 "EXTR_PROPS",
	OpSpreadList:                   "SPREAD_LST",
	OpSpreadTuple:                  "SPREAD_TPL",
	OpAppend:                       "APPEND",
	OpCreateListPattern:            "CRT_LSTP",
	OpCreateObjectPattern:          "CRT_LSTP",
	OpCreateOptionPattern:          "CRT_OPTP",
	OpCreateUnionPattern:           "CRT_UP",
	OpCreateStringUnionPattern:     "CRT_SUP",
	OpCreateRepeatedPatternElement: "CRT_RPE",
	OpCreateSequenceStringPattern:  "CRT_SSP",
	OpCreatePatternNamespace:       "CRT_PNS",
	OpToPattern:                    "TO_PATT",
	OpCreateOptionalPattern:        "CRT_OPTP",
	OpToBool:                       "TO_BOOL",
	OpCreateCheckedString:          "CRT_CSTR",
	OpCreateOption:                 "CRT_OPT",
	OpCreatePath:                   "CRT_PATH",
	OpCreatePathPattern:            "CRT_PATHP",
	OpCreateURL:                    "CRT_URL",
	OpCreateHost:                   "CRT_HST",
	OpCreateRuneRange:              "CRT_RUNERG",
	OpCreateIntRange:               "CRT_INTRG",
	OpCreateUpperBoundRange:        "CRT_UBRG",
	OpCreateTestSuite:              "CRT_TSTS",
	OpCreateTestCase:               "CRT_TSTC",
	OpCreateLifetimeJob:            "CRT_LFJOB",
	OpCreateReceptionHandler:       "CRT_RHANDLER",
	OpSendValue:                    "SEND_VAL",
	OpSpreadObjectPattern:          "SPRD_OBJP",
	BindCapturedLocals:             "BIND_LOCS",
	OpGetGlobal:                    "GET_GLOBAL",
	OpSetGlobal:                    "SET_GLOBAL",
	OpSetMember:                    "SET_MEMBER",
	OpSetIndex:                     "SET_INDEX",
	OpSetSlice:                     "SET_SLICE",
	OpCall:                         "CALL",
	OpReturn:                       "RETURN",
	OpYield:                        "YIELD",
	OpCallPattern:                  "CALL_PATT",
	OpDropPerms:                    "DROP_PERMS",
	OpSpawnRoutine:                 "SPAWN_ROUT",
	OpImport:                       "IMPORT",
	OpGetLocal:                     "GET_LOCAL",
	OpSetLocal:                     "SET_LOCAL",
	OpGetSelf:                      "GET_SELF",
	OpGetSupersys:                  "GET_SUPERSYS",
	OpResolveHost:                  "RSLV_HOST",
	OpAddHostAlias:                 "ADD_HALIAS",
	OpResolvePattern:               "RSLV_PATT",
	OpAddPattern:                   "ADD_PATT",
	OpResolvePatternNamespace:      "RSLV_PNS",
	OpAddPatternNamespace:          "ADD_PATTNS",
	OpPatternNamespaceMemb:         "PNS_MEMB",
	OpIterInit:                     "ITER_INIT",
	OpIterNext:                     "ITER_NEXT",
	OpIterNextChunk:                "ITER_NEXT_CHUNK",
	OpIterKey:                      "ITER_KEY",
	OpIterValue:                    "ITER_VAL",
	OpIterPrune:                    "ITER_PRUNE",
	OpWalkerInit:                   "DWALK_INIT",
	OpIntBin:                       "INT_BIN",
	OpFloatBin:                     "FLOAT_BIN",
	OpNumBin:                       "NUM_BIN",
	OptStrConcat:                   "STR_CONCAT",
	OpConcat:                       "CONCAT",
	OpRange:                        "RANGE",
	OpMemb:                         "MEMB",
	OpDynMemb:                      "DYN_MEMB",
	OpAt:                           "AT",
	OpSlice:                        "SLICE",
	OpAssert:                       "ASSERT",
	OpBlockLock:                    "BLOCK_LOCK",
	OpBlockUnlock:                  "BLOCK_LOCK",
	OpSuspendVM:                    "SUSPEND",
}

// OpcodeOperands is the number of operands.
var OpcodeOperands = [...][]int{
	OpPushConstant:                 {2},
	OpPop:                          {},
	OpCopyTop:                      {},
	OpSwap:                         {},
	OpPushTrue:                     {},
	OpPushFalse:                    {},
	OpEqual:                        {},
	OpNotEqual:                     {},
	OpIs:                           {},
	OpIsNot:                        {},
	OpMinus:                        {},
	OpBooleanNot:                   {},
	OpMatch:                        {},
	OpGroupMatch:                   {2},
	OpIn:                           {},
	OpSubstrOf:                     {},
	OpKeyOf:                        {},
	OpDoSetDifference:              {},
	OpJumpIfFalse:                  {2},
	OpAndJump:                      {2},
	OpOrJump:                       {2},
	OpJump:                         {2},
	OpPushNil:                      {},
	OpGetGlobal:                    {2},
	OpSetGlobal:                    {2},
	OpCreateList:                   {2},
	OpCreateKeyList:                {2},
	OpCreateTuple:                  {2},
	OpCreateObject:                 {2, 2, 2},
	OpCreateRecord:                 {2, 2},
	OpCreateDict:                   {2},
	OpCreateMapping:                {2},
	OpCreateUData:                  {2},
	OpCreateUdataHiearchyEntry:     {2},
	OpSpreadObject:                 {},
	OpExtractProps:                 {2},
	OpSpreadList:                   {},
	OpSpreadTuple:                  {},
	OpAppend:                       {2},
	OpCreateListPattern:            {2, 1},
	OpCreateObjectPattern:          {2, 1},
	OpCreateOptionPattern:          {2},
	OpCreateUnionPattern:           {2},
	OpCreateStringUnionPattern:     {2},
	OpCreateRepeatedPatternElement: {1, 1},
	OpCreateSequenceStringPattern:  {1, 2},
	OpCreatePatternNamespace:       {},
	OpToPattern:                    {},
	OpCreateOptionalPattern:        {},
	OpToBool:                       {},
	OpCreateCheckedString:          {1, 2},
	OpCreateOption:                 {2},
	OpCreatePath:                   {1, 2},
	OpCreatePathPattern:            {1, 2},
	OpCreateURL:                    {2},
	OpCreateHost:                   {2},
	OpCreateRuneRange:              {},
	OpCreateIntRange:               {},
	OpCreateUpperBoundRange:        {},
	OpCreateTestSuite:              {2},
	OpCreateTestCase:               {2},
	OpCreateLifetimeJob:            {2},
	OpCreateReceptionHandler:       {},
	OpSendValue:                    {},
	OpSpreadObjectPattern:          {},
	BindCapturedLocals:             {1},
	OpCall:                         {1, 1, 1},
	OpReturn:                       {1},
	OpYield:                        {1},
	OpCallPattern:                  {1},
	OpDropPerms:                    {},
	OpSpawnRoutine:                 {1, 2, 2},
	OpImport:                       {2},
	OpGetLocal:                     {1},
	OpSetLocal:                     {1},
	OpGetSelf:                      {},
	OpGetSupersys:                  {},
	OpResolveHost:                  {2},
	OpAddHostAlias:                 {2},
	OpResolvePattern:               {2},
	OpAddPattern:                   {2},
	OpResolvePatternNamespace:      {2},
	OpAddPatternNamespace:          {2},
	OpPatternNamespaceMemb:         {2, 2},
	OpSetMember:                    {2},
	OpSetIndex:                     {},
	OpSetSlice:                     {},
	OpIterInit:                     {1},
	OpIterNext:                     {1},
	OpIterNextChunk:                {1},
	OpIterKey:                      {},
	OpIterValue:                    {1},
	OpIterPrune:                    {1},
	OpWalkerInit:                   {},
	OpIntBin:                       {1},
	OpFloatBin:                     {1},
	OpNumBin:                       {1},
	OptStrConcat:                   {},
	OpConcat:                       {1},
	OpRange:                        {1},
	OpMemb:                         {2},
	OpDynMemb:                      {2},
	OpAt:                           {},
	OpSlice:                        {},
	OpAssert:                       {},
	OpBlockLock:                    {1},
	OpBlockUnlock:                  {},
	OpSuspendVM:                    {},
}

// OpcodeOperands is the number of operands.
var OpcodeConstantIndexes = [...][]bool{
	OpPushConstant:                 {true},
	OpPop:                          {},
	OpCopyTop:                      {},
	OpSwap:                         {},
	OpPushTrue:                     {},
	OpPushFalse:                    {},
	OpEqual:                        {},
	OpNotEqual:                     {},
	OpIs:                           {},
	OpIsNot:                        {},
	OpMinus:                        {},
	OpBooleanNot:                   {},
	OpMatch:                        {},
	OpGroupMatch:                   {false},
	OpIn:                           {},
	OpSubstrOf:                     {},
	OpKeyOf:                        {},
	OpDoSetDifference:              {},
	OpJumpIfFalse:                  {false},
	OpAndJump:                      {false},
	OpOrJump:                       {false},
	OpJump:                         {false},
	OpPushNil:                      {},
	OpGetGlobal:                    {true},
	OpSetGlobal:                    {true},
	OpCreateList:                   {false},
	OpCreateKeyList:                {false},
	OpCreateTuple:                  {false},
	OpCreateObject:                 {false, false, true},
	OpCreateRecord:                 {false, false},
	OpCreateDict:                   {false},
	OpCreateMapping:                {true},
	OpCreateUData:                  {false},
	OpCreateUdataHiearchyEntry:     {false},
	OpSpreadObject:                 {},
	OpExtractProps:                 {true},
	OpSpreadList:                   {},
	OpSpreadTuple:                  {},
	OpAppend:                       {false},
	OpCreateListPattern:            {false, false},
	OpCreateObjectPattern:          {false, false},
	OpCreateOptionPattern:          {true},
	OpCreateUnionPattern:           {false},
	OpCreateStringUnionPattern:     {false},
	OpCreateRepeatedPatternElement: {false, false},
	OpCreateSequenceStringPattern:  {false, true},
	OpCreatePatternNamespace:       {},
	OpToPattern:                    {},
	OpCreateOptionalPattern:        {},
	OpToBool:                       {},
	OpCreateCheckedString:          {false, true},
	OpCreateOption:                 {true},
	OpCreatePath:                   {false, true},
	OpCreatePathPattern:            {false, true},
	OpCreateURL:                    {true},
	OpCreateHost:                   {true},
	OpCreateRuneRange:              {},
	OpCreateIntRange:               {},
	OpCreateUpperBoundRange:        {},
	OpCreateTestSuite:              {true},
	OpCreateTestCase:               {true},
	OpCreateLifetimeJob:            {true},
	OpCreateReceptionHandler:       {},
	OpSendValue:                    {},
	OpSpreadObjectPattern:          {},
	BindCapturedLocals:             {false},
	OpCall:                         {false, false, false},
	OpReturn:                       {false},
	OpYield:                        {false},
	OpCallPattern:                  {false},
	OpDropPerms:                    {},
	OpSpawnRoutine:                 {false, true, true},
	OpImport:                       {true},
	OpGetLocal:                     {false},
	OpSetLocal:                     {false},
	OpGetSelf:                      {},
	OpGetSupersys:                  {},
	OpResolveHost:                  {true},
	OpAddHostAlias:                 {true},
	OpResolvePattern:               {true},
	OpAddPattern:                   {true},
	OpResolvePatternNamespace:      {true},
	OpAddPatternNamespace:          {true},
	OpPatternNamespaceMemb:         {true, true},
	OpSetMember:                    {true},
	OpSetIndex:                     {},
	OpSetSlice:                     {},
	OpIterInit:                     {false},
	OpIterNext:                     {false},
	OpIterNextChunk:                {false},
	OpIterKey:                      {},
	OpIterValue:                    {false},
	OpIterPrune:                    {false},
	OpWalkerInit:                   {},
	OpIntBin:                       {false},
	OpFloatBin:                     {false},
	OpNumBin:                       {false},
	OptStrConcat:                   {},
	OpConcat:                       {false},
	OpRange:                        {false},
	OpMemb:                         {true},
	OpDynMemb:                      {true},
	OpAt:                           {},
	OpSlice:                        {},
	OpAssert:                       {},
	OpBlockLock:                    {false},
	OpBlockUnlock:                  {},
	OpSuspendVM:                    {},
}

// ReadOperands reads operands from the bytecode.
func ReadOperands(numOperands []int, ins []byte) (operands []int, offset int) {
	for _, width := range numOperands {
		switch width {
		case 1:
			operands = append(operands, int(ins[offset]))
		case 2:
			operands = append(operands, int(ins[offset+1])|int(ins[offset])<<8)
		}
		offset += width
	}
	return
}

// MakeInstruction returns a bytecode for an opcode and the operands.
func MakeInstruction(opcode Opcode, operands ...int) []byte {
	numOperands := OpcodeOperands[opcode]

	totalLen := 1
	for _, w := range numOperands {
		totalLen += w
	}

	instruction := make([]byte, totalLen)
	instruction[0] = opcode

	offset := 1
	for i, o := range operands {
		width := numOperands[i]
		switch width {
		case 1:
			instruction[offset] = byte(o)
		case 2:
			n := uint16(o)
			instruction[offset] = byte(n >> 8)
			instruction[offset+1] = byte(n)
		}
		offset += width
	}
	return instruction
}

// FormatInstructions returns string representation of bytecode instructions.
func FormatInstructions(ctx *Context, b []byte, posOffset int, leftPadding string, constants []Value) []string {

	var out []string

	fn := func(instr []byte, op Opcode, operands, constantIndexes []int, constants []Value, i int) ([]byte, error) {

		var consts []string

		for _, constant := range constants {
			if constant.HasRepresentation(map[uintptr]int{}, &ReprConfig{}) {
				consts = append(consts, string(GetRepresentation(constant, ctx)))
			} else {
				consts = append(consts, Stringify(constant, nil))
			}
		}

		switch len(operands) {
		case 0:
			out = append(out, fmt.Sprintf("%04d %-10s",
				posOffset+i, OpcodeNames[b[i]]))
		case 1:
			out = append(out, fmt.Sprintf("%04d %-10s %-5d %-5s %-5s %-5s",
				posOffset+i, OpcodeNames[b[i]], operands[0], "", "", ""))
		case 2:
			out = append(out, fmt.Sprintf("%04d %-10s %-5d %-5d %-5s %-5s",
				posOffset+i, OpcodeNames[b[i]],
				operands[0], operands[1], "", ""))
		case 3:
			out = append(out, fmt.Sprintf("%04d %-10s %-5d %-5d %-5d %-5s",
				posOffset+i, OpcodeNames[b[i]],
				operands[0], operands[1], operands[2], ""))
		case 4:
			out = append(out, fmt.Sprintf("%04d %-10s %-5d %-5d %-5d %-5d",
				posOffset+i, OpcodeNames[b[i]],
				operands[0], operands[1], operands[2], operands[3]))
		}
		s := leftPadding + out[len(out)-1]
		if len(consts) >= 1 {
			s += " : " + strings.Join(consts, " ")
		}
		out[len(out)-1] = s
		i += len(instr)

		return nil, nil
	}
	MapInstructions(b, constants, fn)

	return out
}
