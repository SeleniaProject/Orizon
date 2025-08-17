# Orizon Programming Language - 構文仕様書 (EBNF記法)

## 概要

Orizonは、現代的で美しく、かつ強力な構文を持つシステムプログラミング言語です。
C/C++の危険な機能を避けつつ、最大限の表現力を実現します。

## 設計原則

1. **読みやすさ**: 人間が直感的に理解できる構文
2. **一貫性**: 規則に例外を作らない統一的な設計
3. **簡潔性**: 冗長さを排除した美しい表現
4. **安全性**: 危険な操作の構文レベルでの防止
5. **拡張性**: 将来の機能追加に柔軟な構造

## 基本構文要素

### 1. 語彙要素

```ebnf
(* 識別子 - Unicode完全対応 *)
identifier = letter | unicode_letter , { letter | unicode_letter | digit | "_" } ;

(* 文字クラス *)
letter = "a" .. "z" | "A" .. "Z" ;
unicode_letter = (* Unicode Letter category *) ;
digit = "0" .. "9" ;

(* リテラル *)
integer_literal = decimal_literal | binary_literal | hex_literal | octal_literal ;
decimal_literal = "0" | ( "1" .. "9" ) , { digit | "_" } ;
binary_literal = "0b" , binary_digit , { binary_digit | "_" } ;
hex_literal = "0x" , hex_digit , { hex_digit | "_" } ;
octal_literal = "0o" , octal_digit , { octal_digit | "_" } ;

binary_digit = "0" | "1" ;
octal_digit = "0" .. "7" ;
hex_digit = digit | "a" .. "f" | "A" .. "F" ;

(* 浮動小数点リテラル *)
float_literal = decimal_literal , "." , decimal_literal , [ exponent ] |
                decimal_literal , exponent ;
exponent = ( "e" | "E" ) , [ "+" | "-" ] , decimal_literal ;

(* 文字列リテラル *)
string_literal = raw_string | interpreted_string | template_string ;
raw_string = "r" , quote , { unicode_char } , quote ;
interpreted_string = quote , { unicode_char | escape_sequence } , quote ;
template_string = "`" , { template_element } , "`" ;
quote = "\"" ;

(* テンプレート文字列 *)
template_element = template_char | interpolation ;
interpolation = "${" , expression , "}" ;

(* エスケープシーケンス *)
escape_sequence = "\\" , ( "a" | "b" | "f" | "n" | "r" | "t" | "v" | "\\" | "\"" | "'" ) |
                  unicode_escape ;
unicode_escape = "\\u" , hex_digit , hex_digit , hex_digit , hex_digit |
                 "\\U" , hex_digit , hex_digit , hex_digit , hex_digit , 
                         hex_digit , hex_digit , hex_digit , hex_digit ;
```

### 2. 演算子と記号

```ebnf
(* 算術演算子 *)
arithmetic_op = "+" | "-" | "*" | "/" | "%" | "**" ;

(* 比較演算子 *)
comparison_op = "==" | "!=" | "<" | "<=" | ">" | ">=" | "<=>" ;

(* 論理演算子 *)
logical_op = "&&" | "||" | "!" ;

(* ビット演算子 *)
bitwise_op = "&" | "|" | "^" | "<<" | ">>" | "~" ;

(* 代入演算子 *)
assignment_op = "=" | "+=" | "-=" | "*=" | "/=" | "%=" | 
                "&=" | "|=" | "^=" | "<<=" | ">>=" ;

(* その他の演算子 *)
other_op = "?" | ":" | "=>" | "->" | "::" | ".." | "..." | 
           "@" | "#" | "$" | "&" | "|" ;

(* 区切り文字 *)
delimiter = "(" | ")" | "[" | "]" | "{" | "}" | "," | ";" | "." | ":" ;
```

### 3. キーワード

```ebnf
keyword = "as" | "async" | "await" | "break" | "case" | "const" | "continue" |
          "default" | "defer" | "else" | "enum" | "export" | "extern" |
          "false" | "for" | "func" | "if" | "impl" | "import" | "in" |
          "let" | "loop" | "match" | "mut" |
          "return" | "static" | "struct" | "trait" |
          "true" | "type" | "unsafe" | "where" |
          "while" | "yield" ;
```

## プログラム構造

### 1. プログラム全体

```ebnf
program = { item } ;

item = function_declaration |
    struct_declaration |
    enum_declaration |
    trait_declaration |
    impl_block |
    type_alias |
    newtype_declaration |
    const_declaration |
    static_declaration |
    import_declaration |
    export_declaration ;
```

### 2. モジュールシステム

```ebnf
(* インポート宣言 - 文字列やグループ指定は省略し、識別子パスに統一 *)
import_declaration = "import" , import_path , [ "as" , identifier ] ;
import_path = identifier , { "::" , identifier } ;

(* エクスポート宣言 *)
export_declaration = "export" , ( item | "{" , export_list , "}" ) , [ ";" ] ;
export_list = identifier , { "," , identifier } ;
```

## 型システム

### 1. 基本型

```ebnf
(* 型表現 *)
type = basic_type |
    reference_type |
    pointer_type |
    array_type |
    slice_type |
    function_type |
    struct_type |
    enum_type |
    trait_type |
    generic_type |
    dependent_type ;

(* 基本型 *)
basic_type = "i8" | "i16" | "i32" | "i64" | "i128" | "isize" |
             "u8" | "u16" | "u32" | "u64" | "u128" | "usize" |
             "f32" | "f64" | "f128" |
             "bool" | "char" | "string" | "unit" | "never" ;

(* 参照/ポインタ型 *)
reference_type = "&" , [ lifetime ] , [ "mut" ] , type ;
pointer_type = "*" , [ "mut" ] , type ;
lifetime = "'" , identifier ;

(* 配列型 *)
array_type = "[" , type , ";" , const_expression , "]" ;

(* スライス型 *)
slice_type = "[" , type , "]" ;

(* 関数型 *)
function_type = "(" , [ parameter_list ] , ")" , [ "->" , type ] ;
```

### 2. 構造体とenum

```ebnf
(* 構造体宣言 *)
struct_declaration = [ visibility ] , "struct" , identifier , 
                     [ generic_parameters ] , 
                     ( struct_body | ";" ) ;

struct_body = "{" , { field_declaration } , "}" ;
field_declaration = [ visibility ] , identifier , ":" , type , "," ;

(* enum宣言 *)
enum_declaration = [ visibility ] , "enum" , identifier ,
                   [ generic_parameters ] ,
                   "{" , { variant_declaration } , "}" ;

variant_declaration = identifier , [ variant_data ] , "," ;
variant_data = "(" , type_list , ")" | "{" , { field_declaration } , "}" ;

(* newtype 宣言（別名ではなく名義型）*)
(* 実装ではセミコロンは任意（省略可）*)
newtype_declaration = [ visibility ] , "newtype" , identifier , "=" , type , [ ";" ] ;
```

### 3. トレイトシステム

```ebnf
(* トレイト宣言 *)
trait_declaration = [ visibility ] , "trait" , identifier ,
                    [ generic_parameters ] ,
                    [ ":" , trait_bounds ] ,
                    "{" , { trait_item } , "}" ;

trait_item = function_signature |
             associated_type |
             associated_const ;

(* 関連型・関連定数（簡易形）*)
associated_type = "type" , identifier , [ ":" , trait_bounds ] , ";" ;
associated_const = "const" , identifier , ":" , type , ";" ;

(* 実装では 'func' のほか 'fn' も別名として受理する *)
function_signature = ( "func" | "fn" ) , identifier , 
                     [ generic_parameters ] ,
                     "(" , [ parameter_list ] , ")" ,
                     [ "->" , type ] , ";" ;

(* impl ブロック *)
impl_block = "impl" , [ generic_parameters ] , 
             ( type | trait_for_type ) ,
             [ where_clause ] ,
             "{" , { impl_item } , "}" ;

(* impl ブロック内の要素。現行実装では関数宣言のみサポート *)
impl_item = function_declaration ;

trait_for_type = trait_path , "for" , type ;

(* 簡易的なトレイト参照。フルパスを許容 *)
trait_path = identifier , { "::" , identifier } ;
```

## 文と式

### 1. 文

```ebnf
statement = expression_statement |
            let_statement |
            assignment_statement |
            if_statement |
            loop_statement |
            match_statement |
            return_statement |
            break_statement |
            continue_statement |
            defer_statement |
            block_statement ;

(* let文 *)
let_statement = "let" , [ "mut" ] , identifier , 
                [ ":" , type ] , 
                [ "=" , expression ] , ";" ;

(* 代入文 *)
assignment_statement = lvalue , assignment_op , expression , ";" ;

(* ブロック文 *)
block_statement = "{" , { statement } , "}" ;
```

### 2. 制御フロー

```ebnf
(* if文 *)
if_statement = "if" , expression , block_statement ,
               { "else" , "if" , expression , block_statement } ,
               [ "else" , block_statement ] ;

(* ループ文 *)
loop_statement = while_loop | for_loop | loop_infinite ;

while_loop = "while" , expression , block_statement ;

for_loop = "for" , identifier , "in" , expression , block_statement ;

loop_infinite = "loop" , block_statement ;

(* match文 *)
match_statement = "match" , expression , "{" , { match_arm } , "}" ;
match_arm = pattern , [ "if" , expression ] , "=>" , 
            ( expression | block_statement ) , "," ;
```

### 3. 式

```ebnf
expression = literal_expression |
             identifier_expression |
             binary_expression |
             unary_expression |
             call_expression |
             field_access_expression |
             index_expression |
             array_expression |
             struct_expression |
             lambda_expression |
             if_expression |
             match_expression |
             async_expression |
             await_expression ;

(* 二項演算式 *)
binary_expression = expression , binary_op , expression ;

(* 単項演算式 *)
unary_expression = unary_op , expression ;
unary_op = "!" | "-" | "+" | "*" | "&" | "~" ;

(* 関数呼び出し式 *)
call_expression = expression , "(" , [ argument_list ] , ")" ;
argument_list = expression , { "," , expression } ;

(* フィールドアクセス式 *)
field_access_expression = expression , "." , identifier ;

(* インデックス式 *)
index_expression = expression , "[" , expression , "]" ;

(* 配列式 *)
array_expression = "[" , [ expression_list ] , "]" ;
expression_list = expression , { "," , expression } ;

(* 構造体式 *)
struct_expression = type , "{" , [ field_init_list ] , "}" ;
field_init_list = field_init , { "," , field_init } ;
field_init = identifier , ":" , expression ;

(* ラムダ式 *)
lambda_expression = "|" , [ parameter_list ] , "|" , 
                    [ "->" , type ] , 
                    ( expression | block_statement ) ;
```

## 高度な機能

### 1. ジェネリクス

```ebnf
(* ジェネリックパラメータ *)
generic_parameters = "<" , generic_param_list , ">" ;
generic_param_list = generic_param , { "," , generic_param } ;
generic_param = type_param | const_param | lifetime_param ;

type_param = identifier , [ ":" , trait_bounds ] ;
const_param = identifier , ":" , type ;
lifetime_param = lifetime ;

(* トレイト境界 *)
trait_bounds = trait_bound , { "+" , trait_bound } ;
trait_bound = trait_path | lifetime ;

(* where句 *)
where_clause = "where" , where_predicate , { "," , where_predicate } ;
where_predicate = type , ":" , trait_bounds ;
```

### 2. 依存型

```ebnf
(* 依存型 *)
dependent_type = type , "where" , dependent_constraint ;
dependent_constraint = expression ;

(* 例: 配列の長さが型パラメータで制約される *)
(* [T; N] where N > 0 *)
```

### 3. 効果システム

```ebnf
(* 効果注釈 *)
effect_annotation = "effects" , "(" , effect_list , ")" ;
effect_list = effect , { "," , effect } ;
effect = "io" | "alloc" | "unsafe" | "async" | identifier ;

(* 効果付き関数型 *)
function_type_with_effects = "func" , "(" , [ parameter_list ] , ")" ,
                             [ effect_annotation ] ,
                             [ "->" , type ] ;
```

### 4. パターンマッチング

```ebnf
(* パターン *)
pattern = literal_pattern |
          identifier_pattern |
          wildcard_pattern |
          struct_pattern |
          enum_pattern |
          array_pattern |
          slice_pattern |
          or_pattern |
          guard_pattern ;

literal_pattern = literal ;
identifier_pattern = [ "mut" ] , identifier ;
wildcard_pattern = "_" ;

struct_pattern = type , "{" , [ field_pattern_list ] , "}" ;
field_pattern_list = field_pattern , { "," , field_pattern } ;
field_pattern = identifier , [ ":" , pattern ] ;

enum_pattern = type , [ "(" , pattern_list , ")" ] ;
pattern_list = pattern , { "," , pattern } ;

array_pattern = "[" , pattern_list , "]" ;
slice_pattern = "[" , pattern_list , [ ".." , [ pattern ] ] , "]" ;

or_pattern = pattern , "|" , pattern ;
guard_pattern = pattern , "if" , expression ;
```

## メモリ管理

### 1. 所有権システム

```ebnf
(* 所有権注釈 *)
ownership = "owned" | "borrowed" | "shared" ;

(* 参照型の定義は型システム章の reference_type を参照。
    所有権注釈は v1 では構文に含めず予約とし、将来 reference_type に付与可能とする。*)

(* ムーブセマンティクス *)
move_expression = "move" , expression ;
```

### 2. リージョン注釈

```ebnf
(* リージョン宣言 *)
region_declaration = "region" , identifier , "{" , { statement } , "}" ;

(* リージョン型 *)
region_type = type , "@" , identifier ;
```

## コメント

```ebnf
(* コメント *)
comment = line_comment | block_comment | doc_comment ;

line_comment = "//" , { unicode_char } , newline ;
block_comment = "/*" , { unicode_char | newline } , "*/" ;
doc_comment = "///" , { unicode_char } , newline |
              "/**" , { unicode_char | newline } , "*/" ;
```

## サンプルコード

### 基本例

```orizon
// Hello World
func main() {
    print("Hello, Orizon! 🌟")
}

// 型安全な配列アクセス
func safe_access<T, N: usize>(arr: [T; N], index: usize where index < N) -> T {
    arr[index]  // 境界チェック不要
}

// アクターベース並行処理
actor Counter {
    var value: i32 = 0
    
    func increment() -> i32 {
        value += 1
        return value
    }
}

// パターンマッチング
enum Result<T, E> {
    Ok(T),
    Err(E),
}

func handle_result(result: Result<i32, string>) {
    match result {
        Ok(value) => print("Success: {}", value),
        Err(error) => print("Error: {}", error),
    }
}
```

## 構文の特徴

### 1. 美しさと一貫性
- 統一された命名規則
- 予測可能な構文パターン
- 冗長性の排除

### 2. 安全性
- null安全性（Option型）
- 境界チェック（依存型）
- メモリ安全性（所有権システム）

### 3. 表現力
- パターンマッチング
- 関数型プログラミング要素
- メタプログラミング機能

### 4. パフォーマンス
- ゼロコスト抽象化
- コンパイル時計算
- 効率的なメモリレイアウト

この構文仕様は、Orizonが目指す「美しく、安全で、高性能な」プログラミング言語の基盤となります。
