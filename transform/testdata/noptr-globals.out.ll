target datalayout = "e-m:e-p:32:32-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "mips64el-unknown-none-gnuabin32"

@counter = internal global i64 0, section ".noptr"
@table = internal global [4 x i32] [i32 1, i32 2, i32 3, i32 4], section ".noptr"
@pair = internal global { i32, float } zeroinitializer, section ".noptr"
@ptr = internal global ptr null
@withptr = internal global { i32, ptr } zeroinitializer
@arrayofptr = internal global [2 x ptr] zeroinitializer
@const = internal constant i64 7
@placed = internal global i32 0, section ".mine"
@exported = global i32 0
@decl = external global i32
