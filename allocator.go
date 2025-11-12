package tree_sitter

/*
#cgo CFLAGS: -Iinclude -Isrc -std=c11 -D_POSIX_C_SOURCE=200112L -D_DEFAULT_SOURCE
#include <tree_sitter/api.h>
#include "allocator.h"
*/
import "C"

import (
	"sync/atomic"
	"unsafe"
)

var (
	malloc_fn  atomic.Value
	calloc_fn  atomic.Value
	realloc_fn atomic.Value
	free_fn    atomic.Value
)

func init() {
	malloc_fn.Store((func(C.size_t) unsafe.Pointer)(nil))
	calloc_fn.Store((func(C.size_t, C.size_t) unsafe.Pointer)(nil))
	realloc_fn.Store((func(unsafe.Pointer, C.size_t) unsafe.Pointer)(nil))
	free_fn.Store((func(unsafe.Pointer))(nil))
}

//export go_malloc
func go_malloc(size C.size_t) unsafe.Pointer {
	if fn := malloc_fn.Load().(func(C.size_t) unsafe.Pointer); fn != nil {
		return fn(size)
	}
	return C.malloc(size)
}

//export go_calloc
func go_calloc(num, size C.size_t) unsafe.Pointer {
	if fn := calloc_fn.Load().(func(C.size_t, C.size_t) unsafe.Pointer); fn != nil {
		return fn(num, size)
	}
	return C.calloc(num, size)
}

//export go_realloc
func go_realloc(ptr unsafe.Pointer, size C.size_t) unsafe.Pointer {
	if fn := realloc_fn.Load().(func(unsafe.Pointer, C.size_t) unsafe.Pointer); fn != nil {
		return fn(ptr, size)
	}
	return C.realloc(ptr, size)
}

//export go_free
func go_free(ptr unsafe.Pointer) {
	if fn := free_fn.Load().(func(unsafe.Pointer)); fn != nil {
		fn(ptr)
		return
	}
	C.free(ptr)
}

// Sets the memory allocation functions that the core library should use.
func SetAllocator(
	newMalloc func(size uint) unsafe.Pointer,
	newCalloc func(num, size uint) unsafe.Pointer,
	newRealloc func(ptr unsafe.Pointer, size uint) unsafe.Pointer,
	newFree func(ptr unsafe.Pointer),
) {
	if newMalloc == nil && newCalloc == nil && newRealloc == nil && newFree == nil {
		malloc_fn.Store((func(C.size_t) unsafe.Pointer)(nil))
		calloc_fn.Store((func(C.size_t, C.size_t) unsafe.Pointer)(nil))
		realloc_fn.Store((func(unsafe.Pointer, C.size_t) unsafe.Pointer)(nil))
		free_fn.Store((func(unsafe.Pointer))(nil))

		C.ts_set_allocator(nil, nil, nil, nil)
		return
	}

	if newMalloc != nil {
		malloc_fn.Store(func(size C.size_t) unsafe.Pointer {
			return newMalloc(uint(size))
		})
	} else {
		malloc_fn.Store(func(size C.size_t) unsafe.Pointer {
			return C.malloc(size)
		})
	}

	if newCalloc != nil {
		calloc_fn.Store(func(num, size C.size_t) unsafe.Pointer {
			return newCalloc(uint(num), uint(size))
		})
	} else {
		calloc_fn.Store(func(num, size C.size_t) unsafe.Pointer {
			return C.calloc(num, size)
		})
	}

	if newRealloc != nil {
		realloc_fn.Store(func(ptr unsafe.Pointer, size C.size_t) unsafe.Pointer {
			return newRealloc(ptr, uint(size))
		})
	} else {
		realloc_fn.Store(func(ptr unsafe.Pointer, size C.size_t) unsafe.Pointer {
			return C.realloc(ptr, size)
		})
	}

	if newFree != nil {
		free_fn.Store(func(ptr unsafe.Pointer) {
			newFree(ptr)
		})
	} else {
		free_fn.Store(func(ptr unsafe.Pointer) {
			C.free(ptr)
		})
	}

	var cMalloc, cCalloc, cRealloc, cFree unsafe.Pointer
	if newMalloc != nil {
		cMalloc = unsafe.Pointer(C.c_malloc_fn)
	}
	if newCalloc != nil {
		cCalloc = unsafe.Pointer(C.c_calloc_fn)
	}
	if newRealloc != nil {
		cRealloc = unsafe.Pointer(C.c_realloc_fn)
	}
	if newFree != nil {
		cFree = unsafe.Pointer(C.c_free_fn)
	}

	C.ts_set_allocator(
		(*[0]byte)(cMalloc),
		(*[0]byte)(cCalloc),
		(*[0]byte)(cRealloc),
		(*[0]byte)(cFree),
	)
}
