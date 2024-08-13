package tree_sitter

/*
#cgo CFLAGS: -I${SRCDIR}/tree-sitter/lib/include -I${SRCDIR}/tree-sitter/lib/src -std=c11
#include <tree_sitter/api.h>
*/
import "C"

type InputEdit struct {
	StartByte      uint
	OldEndByte     uint
	NewEndByte     uint
	StartPosition  Point
	OldEndPosition Point
	NewEndPosition Point
}

func (i *InputEdit) ToTSInputEdit() *C.TSInputEdit {
	return &C.TSInputEdit{
		start_byte:    C.uint(i.StartByte),
		old_end_byte:  C.uint(i.OldEndByte),
		new_end_byte:  C.uint(i.NewEndByte),
		start_point:   i.StartPosition.toTSPoint(),
		old_end_point: i.OldEndPosition.toTSPoint(),
		new_end_point: i.NewEndPosition.toTSPoint(),
	}
}

func (i *InputEdit) FromTSInputEdit(edit *C.TSInputEdit) {
	i.StartByte = uint(edit.start_byte)
	i.OldEndByte = uint(edit.old_end_byte)
	i.NewEndByte = uint(edit.new_end_byte)
	i.StartPosition.fromTSPoint(edit.start_point)
	i.OldEndPosition.fromTSPoint(edit.old_end_point)
	i.NewEndPosition.fromTSPoint(edit.new_end_point)
}
