(module
  (memory (export "memory") 1)
  (global $arena_offset (mut i32) (i32.const 0))

  (func (export "cabi_realloc")
    (param $old_ptr i32) (param $old_size i32)
    (param $align i32) (param $new_size i32) (result i32)
    (local $ptr i32)
    (local.set $ptr
      (i32.and
        (i32.add (global.get $arena_offset) (i32.sub (local.get $align) (i32.const 1)))
        (i32.xor (i32.sub (local.get $align) (i32.const 1)) (i32.const -1))
      )
    )
    (global.set $arena_offset (i32.add (local.get $ptr) (local.get $new_size)))
    (local.get $ptr)
  )

  (func (export "reset")
    (global.set $arena_offset (i32.const 0))
  )

  ;; contains-sku: returns true only if genre == "zettel" (6 bytes)
  ;; Record layout: genre_ptr(i32), genre_len(i32), ...
  (func (export "contains-sku") (param $record_ptr i32) (result i32)
    (local $genre_ptr i32)
    (local $genre_len i32)

    (local.set $genre_ptr (i32.load (local.get $record_ptr)))
    (local.set $genre_len (i32.load (i32.add (local.get $record_ptr) (i32.const 4))))

    ;; Check length == 6
    (if (i32.ne (local.get $genre_len) (i32.const 6))
      (then (return (i32.const 0)))
    )

    ;; Compare "zettel" byte by byte: z=122 e=101 t=116 t=116 e=101 l=108
    (if (i32.ne (i32.load8_u (local.get $genre_ptr)) (i32.const 122))
      (then (return (i32.const 0))))
    (if (i32.ne (i32.load8_u (i32.add (local.get $genre_ptr) (i32.const 1))) (i32.const 101))
      (then (return (i32.const 0))))
    (if (i32.ne (i32.load8_u (i32.add (local.get $genre_ptr) (i32.const 2))) (i32.const 116))
      (then (return (i32.const 0))))
    (if (i32.ne (i32.load8_u (i32.add (local.get $genre_ptr) (i32.const 3))) (i32.const 116))
      (then (return (i32.const 0))))
    (if (i32.ne (i32.load8_u (i32.add (local.get $genre_ptr) (i32.const 4))) (i32.const 101))
      (then (return (i32.const 0))))
    (if (i32.ne (i32.load8_u (i32.add (local.get $genre_ptr) (i32.const 5))) (i32.const 108))
      (then (return (i32.const 0))))

    (i32.const 1)
  )
)
