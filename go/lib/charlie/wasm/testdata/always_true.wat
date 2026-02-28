(module
  ;; Memory for canonical ABI
  (memory (export "memory") 1)

  ;; Bump allocator state
  (global $arena_offset (mut i32) (i32.const 0))

  ;; cabi_realloc: canonical ABI allocator
  ;; (old_ptr, old_size, align, new_size) -> ptr
  (func (export "cabi_realloc")
    (param $old_ptr i32) (param $old_size i32)
    (param $align i32) (param $new_size i32)
    (result i32)
    (local $ptr i32)
    ;; Align up: ptr = (offset + align - 1) & ~(align - 1)
    (local.set $ptr
      (i32.and
        (i32.add (global.get $arena_offset) (i32.sub (local.get $align) (i32.const 1)))
        (i32.xor (i32.sub (local.get $align) (i32.const 1)) (i32.const -1))
      )
    )
    (global.set $arena_offset (i32.add (local.get $ptr) (local.get $new_size)))
    (local.get $ptr)
  )

  ;; reset: clear arena
  (func (export "reset")
    (global.set $arena_offset (i32.const 0))
  )

  ;; contains-sku: always returns true (1)
  ;; Takes a pointer to a flat SKU record, ignores it
  (func (export "contains-sku") (param $record_ptr i32) (result i32)
    (i32.const 1)
  )
)
