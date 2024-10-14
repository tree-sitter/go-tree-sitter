package tree_sitter

/*
#cgo CFLAGS: -Iinclude -Isrc -std=c11 -D_POSIX_C_SOURCE=200112L -D_DEFAULT_SOURCE
#include <tree_sitter/api.h>
#include <stdio.h>

extern void logCallback(void *payload, TSLogType log_type, char *message);
extern char *readUTF8(void *payload, uint32_t byte_index, TSPoint position, uint32_t *bytes_read);
extern char *readUTF16LE(void *payload, uint32_t byte_offset, TSPoint position, uint32_t *bytes_read);
extern char *readUTF16BE(void *payload, uint32_t byte_offset, TSPoint position, uint32_t *bytes_read);
*/
import "C"

import (
	"context"
	"os"
	"sync/atomic"
	"unsafe"

	"github.com/mattn/go-pointer"
)

// A stateful object that this is used to produce a [Tree] based on some
// source code.
type Parser struct {
	_inner *C.TSParser
}

// Create a new parser.
func NewParser() *Parser {
	return &Parser{_inner: C.ts_parser_new()}
}

func (p *Parser) Close() {
	p.StopPrintingDotGraphs()
	p.SetLogger(nil)
	C.ts_parser_delete(p._inner)
}

// Set the language that the parser should use for parsing.
//
// Returns an error indicating whether or not the language was successfully
// assigned. Nil means assignment succeeded. Non-nil means there was a
// version mismatch: the language was generated with an incompatible
// version of the Tree-sitter CLI. Check the language's version using
// `Version` and compare it to this library's `LANGUAGE_VERSION` and
// `MIN_COMPATIBLE_LANGUAGE_VERSION` constants.
func (p *Parser) SetLanguage(l *Language) error {
	version := l.Version()
	if version >= C.TREE_SITTER_MIN_COMPATIBLE_LANGUAGE_VERSION && version <= C.TREE_SITTER_LANGUAGE_VERSION {
		C.ts_parser_set_language(p._inner, l.Inner)
		return nil
	}
	return &LanguageError{version}
}

// Get the parser's current language.
func (p *Parser) Language() *Language {
	ptr := C.ts_parser_language(p._inner)
	if ptr == nil {
		return nil
	}
	return &Language{Inner: ptr}
}

// A callback that receives log messages during parser.
//
//export logCallback
func logCallback(payload unsafe.Pointer, cLogType C.TSLogType, cMessage *C.char) {
	logger := pointer.Restore(payload).(Logger)
	if logger != nil {
		message := C.GoString(cMessage)
		var logType LogType
		if cLogType == C.TSLogTypeParse {
			logType = LogTypeParse
		} else {
			logType = LogTypeLex
		}
		logger(logType, message)
	}
}

// Set the logging callback that a parser should use during parsing.
func (p *Parser) SetLogger(logger Logger) {
	prevLogger := C.ts_parser_logger(p._inner)
	if prevLogger.payload != nil {
		// Clean up the old logger
		oldLogger := (*Logger)(prevLogger.payload)
		if oldLogger != nil {
			oldLogger = nil
		}
	}

	// Prepare the new logger
	var cLogger C.TSLogger
	if logger != nil {
		cptr := pointer.Save(logger)

		// Set the C logger struct
		cLogger = C.TSLogger{
			payload: cptr,
			log:     (*[0]byte)(C.logCallback),
		}
	} else {
		// Set a null logger if none is provided
		cLogger = C.TSLogger{
			payload: nil,
			log:     nil,
		}
	}

	// Set the new logger in the parser
	C.ts_parser_set_logger(p._inner, cLogger)
}

// Get the parser's current logger.
func (p *Parser) Logger() *Logger {
	logger := C.ts_parser_logger(p._inner)
	return (*Logger)(logger.payload)
}

// Set the destination to which the parser should write debugging graphs
// during parsing. The graphs are formatted in the DOT language. You may
// want to pipe these graphs directly to a `dot(1)` process in order to
// generate SVG output.
func (p *Parser) PrintDotGraphs(file *os.File) {
	C.ts_parser_print_dot_graphs(p._inner, C.int(dupeFD(file.Fd())))
}

// Stop the parser from printing debugging graphs while parsing.
func (p *Parser) StopPrintingDotGraphs() {
	C.ts_parser_print_dot_graphs(p._inner, C.int(-1))
}

// Parse a slice of UTF8 text.
//
// # Arguments:
//   - `text` The UTF8-encoded text to parse.
//   - `old_tree` A previous syntax tree parsed from the same document. If the text of the
//     document has changed since `old_tree` was created, then you must edit `old_tree` to match
//     the new text using [Tree.Edit].
func (p *Parser) Parse(text []byte, oldTree *Tree) *Tree {
	length := len(text)
	return p.ParseWith(func(i int, _ Point) []byte {
		if i < length {
			return text[i:]
		}
		return []byte{}
	}, oldTree)
}

// Parse a slice of UTF8 text.
//
// # Arguments:
//   - `ctx` The context to parse with.
//   - `text` The UTF8-encoded text to parse.
//   - `old_tree` A previous syntax tree parsed from the same document. If the text of the
//     document has changed since `old_tree` was created, then you must edit `old_tree` to match
//     the new text using [Tree.Edit].
func (p *Parser) ParseCtx(ctx context.Context, text []byte, oldTree *Tree) *Tree {
	finish := make(chan struct{})

	if ctx.Done() != nil {
		go func() {
			select {
			case <-ctx.Done():
				atomic.StoreUintptr(p.CancellationFlag(), 1)
			case <-finish:
				return
			}
		}()
	}

	tree := p.Parse(text, oldTree)
	close(finish)

	return tree
}

// Deprecated: Use [Parser.ParseUTF16LE] or [Parser.ParseUTF16BE] instead.
// Parse a slice of UTF16 text.
//
// # Arguments:
//   - `text` The UTF16-encoded text to parse.
//   - `old_tree` A previous syntax tree parsed from the same document. If the text of the
//     document has changed since `old_tree` was created, then you must edit `old_tree` to match
//     the new text using [Tree.Edit].
func (p *Parser) ParseUTF16(text []uint16, oldTree *Tree) *Tree {
	length := len(text)
	return p.ParseUTF16With(func(i int, _ Point) []uint16 {
		if i < length {
			return text[i:]
		}
		return []uint16{}
	}, oldTree)
}

// / Parse a slice of UTF16 little-endian text.
// /
// / # Arguments:
// / * `text` The UTF16-encoded text to parse.
// / * `old_tree` A previous syntax tree parsed from the same document. If the text of the
// /   document has changed since `old_tree` was created, then you must edit `old_tree` to match
// /   the new text using [Tree.Edit].
func (p *Parser) ParseUTF16LE(text []uint16, oldTree *Tree) *Tree {
	length := len(text)
	return p.ParseUTF16LEWith(func(i int, _ Point) []uint16 {
		if i < length {
			return text[i:]
		}
		return []uint16{}
	}, oldTree)
}

// / Parse a slice of UTF16 big-endian text.
// /
// / # Arguments:
// / * `text` The UTF16-encoded text to parse.
// / * `old_tree` A previous syntax tree parsed from the same document. If the text of the
// /   document has changed since `old_tree` was created, then you must edit `old_tree` to match
// /   the new text using [Tree.Edit].
func (p *Parser) ParseUTF16BE(text []uint16, oldTree *Tree) *Tree {
	length := len(text)
	return p.ParseUTF16BEWith(func(i int, _ Point) []uint16 {
		if i < length {
			return text[i:]
		}
		return []uint16{}
	}, oldTree)
}

type payload[T any] struct {
	callback func(int, Point) []T
	text     []T
	cStrings []*C.char
}

// This C function is passed to Tree-sitter as the input callback.
//
//export readUTF8
func readUTF8(_payload unsafe.Pointer, byteIndex C.uint32_t, position C.TSPoint, bytesRead *C.uint32_t) *C.char {
	payload := pointer.Restore(_payload).(*payload[byte])
	payload.text = payload.callback(int(byteIndex), Point{uint(position.row), uint(position.column)})
	*bytesRead = C.uint32_t(len(payload.text))
	strbytes := C.CString(string(payload.text))
	payload.cStrings = append(payload.cStrings, strbytes)
	return strbytes
}

// Parse UTF8 text provided in chunks by a callback.
//
// # Arguments:
//   - `callback` A function that takes a byte offset and position and returns a slice of
//     UTF8-encoded text starting at that byte offset and position. The slices can be of any
//     length. If the given position is at the end of the text, the callback should return an
//     empty slice.
//   - `old_tree` A previous syntax tree parsed from the same document. If the text of the
//     document has changed since `old_tree` was created, then you must edit `old_tree` to match
//     the new text using [Tree.Edit].
func (p *Parser) ParseWith(callback func(int, Point) []byte, oldTree *Tree) *Tree {
	payload := payload[byte]{
		callback: callback,
		text:     nil,
		cStrings: make([]*C.char, 0),
	}

	defer func() {
		for _, cString := range payload.cStrings {
			go_free(unsafe.Pointer(cString))
		}
	}()

	cptr := pointer.Save(&payload)
	defer pointer.Unref(cptr)

	cInput := C.TSInput{
		payload:  unsafe.Pointer(cptr),
		read:     (*[0]byte)(C.readUTF8),
		encoding: C.TSInputEncodingUTF8,
	}

	var cOldTree *C.TSTree
	if oldTree != nil {
		cOldTree = oldTree._inner
	}

	cNewTree := C.ts_parser_parse(p._inner, cOldTree, cInput)

	if cNewTree != nil {
		return newTree(cNewTree)
	}

	return nil
}

func cStringUTF16(s []uint16) *C.char {
	if len(s)+1 <= 0 {
		panic("string too large")
	}
	p := _cgo_cmalloc(uint64((len(s) + 1) * 2))
	sliceHeader := struct {
		p   unsafe.Pointer
		len int
		cap int
	}{p, len(s) + 1, len(s) + 1}
	b := *(*[]uint16)(unsafe.Pointer(&sliceHeader))
	copy(b, s)
	b[len(s)] = 0
	return (*C.char)(p)
}

// This C function is passed to Tree-sitter as the input callback.
//
//export readUTF16LE
func readUTF16LE(_payload unsafe.Pointer, byteOffset uint32, position C.TSPoint, bytesRead *uint32) *C.char {
	payload := pointer.Restore(_payload).(*payload[uint16])
	payload.text = payload.callback(int(byteOffset/2), Point{uint(position.row), uint(position.column / 2)})
	*bytesRead = uint32(len(payload.text) * 2)
	strbytes := cStringUTF16(payload.text)
	payload.cStrings = append(payload.cStrings, strbytes)
	return strbytes
}

// This C function is passed to Tree-sitter as the input callback.
//
//export readUTF16BE
func readUTF16BE(_payload unsafe.Pointer, byteOffset uint32, position C.TSPoint, bytesRead *uint32) *C.char {
	payload := pointer.Restore(_payload).(*payload[uint16])
	payload.text = payload.callback(int(byteOffset/2), Point{uint(position.row), uint(position.column / 2)})
	*bytesRead = uint32(len(payload.text) * 2)
	strbytes := cStringUTF16(payload.text)
	payload.cStrings = append(payload.cStrings, strbytes)
	return strbytes
}

// Deprecated: Use [Parser.ParseUTF16LEWith] or [Parser.ParseUTF16BEWith] instead.
// Parse UTF16 text provided in chunks by a callback.
//
// # Arguments:
//   - `callback` A function that takes a code point offset and position and returns a slice of
//     UTF16-encoded text starting at that byte offset and position. The slices can be of any
//     length. If the given position is at the end of the text, the callback should return an
//     empty slice.
//   - `old_tree` A previous syntax tree parsed from the same document. If the text of the
//     document has changed since `old_tree` was created, then you must edit `old_tree` to match
//     the new text using [Tree.Edit].
func (p *Parser) ParseUTF16With(callback func(int, Point) []uint16, oldTree *Tree) *Tree {
	return p.ParseUTF16LEWith(callback, oldTree)
}

// Parse UTF16 little-endian text provided in chunks by a callback.
//
// # Arguments:
//   - `callback` A function that takes a code point offset and position and returns a slice of
//     UTF16-encoded text starting at that byte offset and position. The slices can be of any
//     length. If the given position is at the end of the text, the callback should return an
//     empty slice.
//   - `old_tree` A previous syntax tree parsed from the same document. If the text of the
//     document has changed since `old_tree` was created, then you must edit `old_tree` to match
//     the new text using [Tree.Edit].
func (p *Parser) ParseUTF16LEWith(callback func(int, Point) []uint16, oldTree *Tree) *Tree {
	payload := payload[uint16]{
		callback: callback,
		text:     nil,
		cStrings: make([]*C.char, 0),
	}

	defer func() {
		for _, cString := range payload.cStrings {
			go_free(unsafe.Pointer(cString))
		}
	}()

	cptr := pointer.Save(&payload)
	defer pointer.Unref(cptr)

	cInput := C.TSInput{
		payload:  unsafe.Pointer(cptr),
		read:     (*[0]byte)(C.readUTF16LE),
		encoding: C.TSInputEncodingUTF16LE,
	}

	var cOldTree *C.TSTree
	if oldTree != nil {
		cOldTree = oldTree._inner
	}

	cNewTree := C.ts_parser_parse(p._inner, cOldTree, cInput)

	if cNewTree != nil {
		return newTree(cNewTree)
	}

	return nil
}

// Parse UTF16 big-endian text provided in chunks by a callback.
//
// # Arguments:
//   - `callback` A function that takes a code point offset and position and returns a slice of
//     UTF16-encoded text starting at that byte offset and position. The slices can be of any
//     length. If the given position is at the end of the text, the callback should return an
//     empty slice.
//   - `old_tree` A previous syntax tree parsed from the same document. If the text of the
//     document has changed since `old_tree` was created, then you must edit `old_tree` to match
//     the new text using [Tree.Edit].
func (p *Parser) ParseUTF16BEWith(callback func(int, Point) []uint16, oldTree *Tree) *Tree {
	payload := payload[uint16]{
		callback: callback,
		text:     nil,
		cStrings: make([]*C.char, 0),
	}

	defer func() {
		for _, cString := range payload.cStrings {
			go_free(unsafe.Pointer(cString))
		}
	}()

	cptr := pointer.Save(&payload)
	defer pointer.Unref(cptr)

	cInput := C.TSInput{
		payload:  unsafe.Pointer(cptr),
		read:     (*[0]byte)(C.readUTF16BE),
		encoding: C.TSInputEncodingUTF16BE,
	}

	var cOldTree *C.TSTree
	if oldTree != nil {
		cOldTree = oldTree._inner
	}

	cNewTree := C.ts_parser_parse(p._inner, cOldTree, cInput)

	if cNewTree != nil {
		return newTree(cNewTree)
	}

	return nil
}

// Instruct the parser to start the next parse from the beginning.
//
// If the parser previously failed because of a timeout or a cancellation,
// then by default, it will resume where it left off on the next call
// to [Parser.Parse] or other parsing functions. If you don't
// want to resume, and instead intend to use this parser to parse some
// other document, you must call `Reset` first.
func (p *Parser) Reset() {
	C.ts_parser_reset(p._inner)
}

// Get the duration in microseconds that parsing is allowed to take.
//
// This is set via [Parser.SetTimeoutMicros].
func (p *Parser) TimeoutMicros() uint64 {
	return uint64(C.ts_parser_timeout_micros(p._inner))
}

// Set the maximum duration in microseconds that parsing should be allowed
// to take before halting.
//
// If parsing takes longer than this, it will halt early, returning `nil`.
// See [Parser.Parse] for more information.
func (p *Parser) SetTimeoutMicros(timeoutMicros uint64) {
	C.ts_parser_set_timeout_micros(p._inner, C.uint64_t(timeoutMicros))
}

// Get the ranges of text that the parser will include when parsing.
func (p *Parser) IncludedRanges() []Range {
	var count C.uint
	ptr := C.ts_parser_included_ranges(p._inner, &count)
	ranges := make([]Range, int(count))
	for i := uintptr(0); i < uintptr(count); i++ {
		val := *(*C.TSRange)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + i*unsafe.Sizeof(*ptr)))
		ranges[i] = Range{
			StartByte:  uint(val.start_byte),
			EndByte:    uint(val.end_byte),
			StartPoint: Point{Row: uint(val.start_point.row), Column: uint(val.start_point.column)},
			EndPoint:   Point{Row: uint(val.end_point.row), Column: uint(val.end_point.column)},
		}
	}
	return ranges
}

// Set the ranges of text that the parser should include when parsing.
//
// By default, the parser will always include entire documents. This
// function allows you to parse only a *portion* of a document but
// still return a syntax tree whose ranges match up with the document
// as a whole. You can also pass multiple disjoint ranges.
//
// If `ranges` is empty, then the entire document will be parsed.
// Otherwise, the given ranges must be ordered from earliest to latest
// in the document, and they must not overlap. That is, the following
// must hold for all `i` < `length - 1`:
// ```text
//
//	ranges[i].end_byte <= ranges[i + 1].start_byte
//
// ```
// If this requirement is not satisfied, method will return
// [IncludedRangesError] error with an offset in the passed ranges
// slice pointing to a first incorrect range.
func (p *Parser) SetIncludedRanges(ranges []Range) error {
	tsRanges := make([]C.TSRange, len(ranges))
	for i, r := range ranges {
		tsRanges[i] = C.TSRange{
			start_byte:  C.uint32_t(r.StartByte),
			end_byte:    C.uint32_t(r.EndByte),
			start_point: r.StartPoint.toTSPoint(),
			end_point:   r.EndPoint.toTSPoint(),
		}
	}
	var cPtr *C.TSRange
	if len(tsRanges) > 0 {
		cPtr = &tsRanges[0]
	}
	result := C.ts_parser_set_included_ranges(p._inner, cPtr, C.uint32_t(len(tsRanges)))
	if result {
		return nil
	}
	var prevEndByte uint
	for i, r := range ranges {
		if r.StartByte < prevEndByte || r.EndByte < r.StartByte {
			return &IncludedRangesError{uint32(i)}
		}
		prevEndByte = r.EndByte
	}
	return &IncludedRangesError{0}
}

// Get the parser's current cancellation flag pointer.
func (p *Parser) CancellationFlag() *uintptr {
	return (*uintptr)(unsafe.Pointer(C.ts_parser_cancellation_flag(p._inner)))
}

// Set the parser's current cancellation flag pointer.
//
// If a pointer is assigned, then the parser will periodically read from
// this pointer during parsing. If it reads a non-zero value, it will halt
// early, returning `nil`. See [Parser.Parse] for more
// information.
func (p *Parser) SetCancellationFlag(flag *uintptr) {
	C.ts_parser_set_cancellation_flag(p._inner, (*C.size_t)(unsafe.Pointer(flag)))
}
