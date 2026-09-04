target datalayout = "e-m:e-p:32:32-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "mips64el-unknown-none-gnuabin32"

@counter = internal global i64 0, section ".noptr"
@ptr = internal global ptr null, section ".scan"
@withptr = internal global { i32, ptr } zeroinitializer, section ".scan"
@const = internal constant i64 7
@placed = internal global i32 0, section ".mine"
@exported = global i32 0, section ".scan"
@decl = external global i32
@llvm.used = appending global [1 x ptr] [ptr @exported]
