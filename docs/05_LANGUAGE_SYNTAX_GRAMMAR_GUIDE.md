# Orizon Programming Language - 言語構文・文法完全ガイド

## 概要

Orizonプログラミング言語の構文と文法について、EBNF記法を用いた正式な仕様から実用的なサンプルコードまで、包括的に解説します。言語学習者から言語実装者まで、すべての読者が参照できる完全なリファレンスです。

---

## 設計哲学と構文原則

### 設計哲学
1. **読みやすさ優先**: 人間が直感的に理解できる美しい構文
2. **一貫性の徹底**: 規則に例外を作らない統一的な設計
3. **簡潔性の追求**: 冗長さを排除した表現力豊かな記述
4. **安全性の保証**: 危険な操作の構文レベルでの防止
5. **拡張性の確保**: 将来の機能追加に柔軟な構造

### 構文の特徴
- **型推論の活用**: 明示的型注釈と推論のバランス
- **パターンマッチング**: 強力で安全なデータ分解
- **所有権システム**: Rustライクな安全性とC++ライクな表現力
- **非同期ファーストクラス**: async/await の自然な統合
- **マクロシステム**: 衛生的で安全なコード生成

---

## 基本構文要素

### 語彙要素（Lexical Elements）

#### 識別子（Identifiers）
```ebnf
identifier = letter | unicode_letter , { letter | unicode_letter | digit | "_" } ;
letter = "a" .. "z" | "A" .. "Z" ;
unicode_letter = (* Unicode Letter category Ll, Lu, Lt, Lo, Nl *) ;
digit = "0" .. "9" ;
```

**有効な識別子の例**:
```orizon
// 基本的な識別子
variable_name
functionName
TypeName
CONSTANT_VALUE

// Unicode識別子（国際化対応）
変数名        // 日本語
переменная    // ロシア語
变量         // 中国語
متغير        // アラビア語

// プライベート識別子（アンダースコア開始）
_private_var
_internal_function
__system_reserved
```

#### リテラル（Literals）

##### 整数リテラル
```ebnf
integer_literal = decimal_literal | binary_literal | hex_literal | octal_literal ;
decimal_literal = "0" | ( "1" .. "9" ) , { digit | "_" } ;
binary_literal = "0b" , binary_digit , { binary_digit | "_" } ;
hex_literal = "0x" , hex_digit , { hex_digit | "_" } ;
octal_literal = "0o" , octal_digit , { octal_digit | "_" } ;
```

**整数リテラルの例**:
```orizon
// 10進数
42
1_000_000
0

// 2進数
0b1010_1111
0b1111_0000_1010_1010

// 16進数
0xFF
0x1234_ABCD
0xDEAD_BEEF

// 8進数
0o755
0o644
```

##### 浮動小数点リテラル
```ebnf
float_literal = decimal_literal , "." , decimal_literal , [ exponent ] |
                decimal_literal , exponent ;
exponent = ( "e" | "E" ) , [ "+" | "-" ] , decimal_literal ;
```

**浮動小数点リテラルの例**:
```orizon
// 基本的な浮動小数点数
3.14159
2.718281828
0.5

// 指数表記
1.23e10
4.56E-7
1e100

// アンダースコア区切り
3.141_592_653
6.022_140_76e23
```

##### 文字列リテラル
```ebnf
string_literal = raw_string | interpreted_string | template_string ;
raw_string = "r" , quote , { unicode_char } , quote ;
interpreted_string = quote , { unicode_char | escape_sequence } , quote ;
template_string = "`" , { template_element } , "`" ;
quote = "\"" ;
```

**文字列リテラルの例**:
```orizon
// 通常の文字列
"Hello, Orizon!"
"Unicode support: 🚀 世界 🌍"

// Raw文字列（エスケープなし）
r"C:\Windows\System32\kernel32.dll"
r"正規表現: \d+\.\d+"

// テンプレート文字列（文字列補間）
let name = "World";
let message = `Hello, ${name}!`;
let complex = `結果: ${x + y}, 時刻: ${now()}`;

// 複数行文字列
let multiline = `
    これは複数行の
    文字列です。
    インデントも保持されます。
`;
```

##### エスケープシーケンス
```ebnf
escape_sequence = "\\" , ( "a" | "b" | "f" | "n" | "r" | "t" | "v" | "\\" | "\"" | "'" ) |
                  unicode_escape ;
unicode_escape = "\\u" , hex_digit , hex_digit , hex_digit , hex_digit |
                 "\\U" , hex_digit , hex_digit , hex_digit , hex_digit , 
                         hex_digit , hex_digit , hex_digit , hex_digit ;
```

**エスケープシーケンスの例**:
```orizon
// 標準エスケープ
"改行\n文字列"
"タブ\t区切り"
"引用符: \"Hello\""
"バックスラッシュ: \\"

// Unicodeエスケープ
"Unicode: \u03B1\u03B2\u03B3"  // αβγ
"Emoji: \U0001F680"           // 🚀
```

#### 文字リテラル
```ebnf
char_literal = "'" , ( unicode_char | escape_sequence ) , "'" ;
```

**文字リテラルの例**:
```orizon
'a'
'α'
'🌟'
'\n'
'\u03B1'  // α
```

### 演算子（Operators）

#### 算術演算子
```ebnf
arithmetic_op = "+" | "-" | "*" | "/" | "%" | "**" ;
```

**算術演算子の例**:
```orizon
let a = 10 + 5;      // 加算: 15
let b = 10 - 3;      // 減算: 7
let c = 4 * 6;       // 乗算: 24
let d = 15 / 3;      // 除算: 5
let e = 17 % 5;      // 剰余: 2
let f = 2 ** 8;      // 累乗: 256
```

#### 比較演算子
```ebnf
comparison_op = "==" | "!=" | "<" | "<=" | ">" | ">=" | "<=>" ;
```

**比較演算子の例**:
```orizon
let eq = (5 == 5);    // 等価: true
let ne = (3 != 7);    // 非等価: true
let lt = (2 < 8);     // 未満: true
let le = (4 <= 4);    // 以下: true
let gt = (9 > 3);     // 超過: true
let ge = (6 >= 6);    // 以上: true
let cmp = (a <=> b);  // 三項比較: -1, 0, 1
```

#### 論理演算子
```ebnf
logical_op = "&&" | "||" | "!" ;
```

**論理演算子の例**:
```orizon
let and_result = true && false;   // 論理積: false
let or_result = true || false;    // 論理和: true
let not_result = !true;           // 論理否定: false

// 短絡評価
let safe = (ptr != null) && (ptr.is_valid());
```

#### ビット演算子
```ebnf
bitwise_op = "&" | "|" | "^" | "<<" | ">>" | "~" ;
```

**ビット演算子の例**:
```orizon
let bit_and = 0b1100 & 0b1010;   // ビット積: 0b1000
let bit_or = 0b1100 | 0b1010;    // ビット和: 0b1110
let bit_xor = 0b1100 ^ 0b1010;   // 排他的論理和: 0b0110
let left_shift = 5 << 2;         // 左シフト: 20
let right_shift = 20 >> 2;       // 右シフト: 5
let bit_not = ~0b1010;           // ビット否定: 0b0101
```

#### 代入演算子
```ebnf
assignment_op = "=" | "+=" | "-=" | "*=" | "/=" | "%=" | 
                "&=" | "|=" | "^=" | "<<=" | ">>=" ;
```

**代入演算子の例**:
```orizon
let mut x = 10;
x = 20;        // 代入
x += 5;        // 加算代入: x = x + 5
x -= 3;        // 減算代入: x = x - 3
x *= 2;        // 乗算代入: x = x * 2
x /= 4;        // 除算代入: x = x / 4
x %= 7;        // 剰余代入: x = x % 7

let mut flags = 0b1010;
flags &= 0b1100;   // ビット積代入
flags |= 0b0011;   // ビット和代入
flags ^= 0b1111;   // 排他的論理和代入
flags <<= 2;       // 左シフト代入
flags >>= 1;       // 右シフト代入
```

#### その他の演算子
```ebnf
other_op = "?" | ":" | "=>" | "->" | "::" | ".." | "..." | 
           "@" | "#" | "$" | "&" | "|" ;
```

**その他の演算子の例**:
```orizon
// 三項演算子
let result = condition ? value_if_true : value_if_false;

// 関数型アロー
let closure = |x: i32| -> i32 { x * 2 };

// パス区切り
let qualified_name = module::function;

// 範囲演算子
let range1 = 0..10;      // 0から9まで
let range2 = 0..=10;     // 0から10まで（境界含む）
let range3 = ..=5;       // 5以下
let range4 = 3..;        // 3以上

// 可変長引数
func variadic(args: ...i32) { /* ... */ }

// アドレス取得
let reference = &variable;

// 参照外し
let value = *reference;
```

### キーワード（Keywords）

```ebnf
keyword = "as" | "async" | "await" | "break" | "case" | "const" | "continue" |
          "default" | "defer" | "else" | "enum" | "export" | "extern" |
          "false" | "for" | "func" | "if" | "impl" | "import" | "in" |
          "let" | "loop" | "match" | "mut" |
          "return" | "static" | "struct" | "trait" |
          "true" | "type" | "unsafe" | "where" |
          "while" | "yield" ;
```

---

## 型システム

### 基本型（Primitive Types）

#### 整数型
```orizon
// 符号付き整数
let i8_var: i8 = -128;
let i16_var: i16 = -32768;
let i32_var: i32 = -2147483648;
let i64_var: i64 = -9223372036854775808;
let i128_var: i128 = -170141183460469231731687303715884105728;
let isize_var: isize = -1;  // プラットフォーム依存

// 符号なし整数
let u8_var: u8 = 255;
let u16_var: u16 = 65535;
let u32_var: u32 = 4294967295;
let u64_var: u64 = 18446744073709551615;
let u128_var: u128 = 340282366920938463463374607431768211455;
let usize_var: usize = 100;  // プラットフォーム依存
```

#### 浮動小数点型
```orizon
let f32_var: f32 = 3.14159;
let f64_var: f64 = 2.718281828459045;
```

#### 文字・文字列型
```orizon
let char_var: char = 'α';
let string_var: String = "Hello, World!";
let str_slice: &str = "文字列スライス";
```

#### 論理型
```orizon
let bool_var: bool = true;
let false_var: bool = false;
```

### 複合型（Compound Types）

#### 配列型
```ebnf
array_type = "[" , type , ";" , integer_literal , "]" ;
```

**配列型の例**:
```orizon
// 固定長配列
let numbers: [i32; 5] = [1, 2, 3, 4, 5];
let zeros: [i32; 100] = [0; 100];  // 全要素を0で初期化

// 配列アクセス
let first = numbers[0];
let last = numbers[numbers.len() - 1];

// 多次元配列
let matrix: [[i32; 3]; 3] = [
    [1, 2, 3],
    [4, 5, 6],
    [7, 8, 9]
];
```

#### タプル型
```ebnf
tuple_type = "(" , [ type , { "," , type } , [ "," ] ] , ")" ;
```

**タプル型の例**:
```orizon
// 基本的なタプル
let point: (f64, f64) = (3.14, 2.71);
let person: (String, i32, bool) = ("Alice".to_string(), 30, true);

// タプル分解
let (x, y) = point;
let (name, age, is_active) = person;

// ユニット型（空のタプル）
let unit: () = ();

// ネストしたタプル
let nested: ((i32, i32), (String, String)) = ((1, 2), ("a".to_string(), "b".to_string()));
```

#### スライス型
```ebnf
slice_type = "[" , type , "]" ;
```

**スライス型の例**:
```orizon
// スライスの作成
let array = [1, 2, 3, 4, 5];
let slice: &[i32] = &array[1..4];  // [2, 3, 4]

// 可変スライス
let mut mut_array = [1, 2, 3, 4, 5];
let mut_slice: &mut [i32] = &mut mut_array[..];

// 動的配列（Vec）
let mut vector: Vec<i32> = vec![1, 2, 3];
vector.push(4);
let vec_slice: &[i32] = &vector;
```

### ポインタ・参照型

#### 参照型
```ebnf
reference_type = "&" , [ "mut" ] , type ;
```

**参照型の例**:
```orizon
let value = 42;
let immutable_ref: &i32 = &value;      // 不変参照
let mut mutable_value = 100;
let mutable_ref: &mut i32 = &mut mutable_value;  // 可変参照

// 参照の使用
println!("Value: {}", *immutable_ref);
*mutable_ref += 10;
```

#### 生ポインタ型
```ebnf
pointer_type = "*" , [ "const" | "mut" ] , type ;
```

**生ポインタ型の例**:
```orizon
unsafe {
    let value = 42;
    let const_ptr: *const i32 = &value as *const i32;
    let mut_value = 100;
    let mut_ptr: *mut i32 = &mut mut_value as *mut i32;
    
    // 生ポインタの参照外し（unsafeブロック内のみ）
    let dereferenced = *const_ptr;
    *mut_ptr = 200;
}
```

### 関数型
```ebnf
function_type = "func" , "(" , [ parameter_list ] , ")" , [ "->" , type ] ;
```

**関数型の例**:
```orizon
// 関数ポインタ
let add_func: func(i32, i32) -> i32 = |x, y| x + y;

// 高階関数
func apply_twice(f: func(i32) -> i32, x: i32) -> i32 {
    f(f(x))
}

let double = |x| x * 2;
let result = apply_twice(double, 5);  // 20
```

---

## 変数宣言

### let文
```ebnf
let_statement = "let" , [ "mut" ] , pattern , [ ":" , type ] , [ "=" , expression ] , ";" ;
```

**let文の例**:
```orizon
// 基本的な変数宣言
let x = 42;                    // 型推論
let y: i32 = 100;              // 明示的型指定
let z: i32;                    // 宣言のみ（後で初期化）

// 可変変数
let mut counter = 0;
counter += 1;

// パターンマッチング付き宣言
let (x, y) = (10, 20);
let [first, second, ..rest] = [1, 2, 3, 4, 5];

// 構造体分解
struct Point { x: f64, y: f64 }
let Point { x, y } = Point { x: 3.0, y: 4.0 };
```

### const文
```ebnf
const_statement = "const" , identifier , ":" , type , "=" , expression , ";" ;
```

**const文の例**:
```orizon
// グローバル定数
const PI: f64 = 3.141592653589793;
const MAX_SIZE: usize = 1024;

// 複合型の定数
const DEFAULT_POINT: Point = Point { x: 0.0, y: 0.0 };
const FIBONACCI: [u64; 10] = [1, 1, 2, 3, 5, 8, 13, 21, 34, 55];
```

### static文
```ebnf
static_statement = "static" , [ "mut" ] , identifier , ":" , type , "=" , expression , ";" ;
```

**static文の例**:
```orizon
// 静的変数
static COUNTER: AtomicUsize = AtomicUsize::new(0);

// 可変静的変数（unsafe）
static mut GLOBAL_STATE: i32 = 0;

unsafe {
    GLOBAL_STATE = 42;
}
```

---

## 関数定義

### 関数定義構文
```ebnf
function_declaration = [ "async" ] , "func" , identifier , 
                       [ generic_parameters ] ,
                       "(" , [ parameter_list ] , ")" ,
                       [ "->" , type ] ,
                       [ where_clause ] ,
                       block_statement ;

parameter_list = parameter , { "," , parameter } , [ "," ] ;
parameter = [ "mut" ] , identifier , ":" , type ;
```

**関数定義の例**:
```orizon
// 基本的な関数
func add(a: i32, b: i32) -> i32 {
    return a + b;
}

// 戻り値なし（ユニット型）
func print_hello() {
    println!("Hello, Orizon!");
}

// 式ベースの戻り値（return省略可能）
func multiply(x: i32, y: i32) -> i32 {
    x * y  // 最後の式が戻り値
}

// 可変引数
func print_numbers(numbers: ...i32) {
    for num in numbers {
        println!("{}", num);
    }
}

// ジェネリック関数
func swap<T>(a: &mut T, b: &mut T) {
    let temp = std::mem::replace(a, std::mem::replace(b, temp));
}

// 非同期関数
async func fetch_data(url: &str) -> Result<String, Error> {
    let response = http_client.get(url).await?;
    response.text().await
}
```

### クロージャ
```ebnf
closure_expression = "|" , [ parameter_list ] , "|" , [ "->" , type ] , expression ;
```

**クロージャの例**:
```orizon
// 基本的なクロージャ
let add_one = |x| x + 1;
let result = add_one(5);  // 6

// 型注釈付きクロージャ
let multiply: func(i32, i32) -> i32 = |x: i32, y: i32| -> i32 { x * y };

// 環境キャプチャ
let multiplier = 10;
let scale = |x| x * multiplier;  // multiplierをキャプチャ

// 可変キャプチャ
let mut counter = 0;
let mut increment = || {
    counter += 1;
    counter
};
```

---

## 制御構造

### 条件分岐（if文）
```ebnf
if_statement = "if" , expression , block_statement , 
               { "else" , "if" , expression , block_statement } ,
               [ "else" , block_statement ] ;
```

**if文の例**:
```orizon
// 基本的なif文
if x > 0 {
    println!("正の数");
}

// if-else
if x > 0 {
    println!("正の数");
} else {
    println!("0以下");
}

// if-else if-else
if x > 0 {
    println!("正の数");
} else if x < 0 {
    println!("負の数");
} else {
    println!("ゼロ");
}

// if式（値を返す）
let message = if x > 0 { "positive" } else { "non-positive" };

// パターンマッチングとの組み合わせ
if let Some(value) = optional_value {
    println!("値: {}", value);
}
```

### ループ構造

#### while文
```ebnf
while_statement = "while" , expression , block_statement ;
```

**while文の例**:
```orizon
// 基本的なwhile文
let mut i = 0;
while i < 10 {
    println!("{}", i);
    i += 1;
}

// 条件付きwhile
let mut input = get_input();
while !input.is_empty() {
    process(input);
    input = get_input();
}
```

#### for文
```ebnf
for_statement = "for" , pattern , "in" , expression , block_statement ;
```

**for文の例**:
```orizon
// 基本的なfor文
for i in 0..10 {
    println!("{}", i);
}

// 配列のイテレーション
let numbers = [1, 2, 3, 4, 5];
for num in &numbers {
    println!("{}", num);
}

// インデックス付きイテレーション
for (index, value) in numbers.iter().enumerate() {
    println!("{}番目: {}", index, value);
}

// パターンマッチング付きfor文
let pairs = vec![(1, 2), (3, 4), (5, 6)];
for (x, y) in pairs {
    println!("({}, {})", x, y);
}
```

#### loop文
```ebnf
loop_statement = "loop" , block_statement ;
```

**loop文の例**:
```orizon
// 無限ループ
loop {
    let input = get_input();
    if input == "quit" {
        break;
    }
    process_input(input);
}

// ラベル付きループ
'outer: loop {
    loop {
        if some_condition() {
            break 'outer;  // 外側のループを抜ける
        }
        if other_condition() {
            continue 'outer;  // 外側のループの次の繰り返しへ
        }
    }
}

// 値を返すloop
let result = loop {
    let value = calculate();
    if value > threshold {
        break value;  // valueを返してループ終了
    }
};
```

### パターンマッチング（match文）
```ebnf
match_statement = "match" , expression , "{" , { match_arm } , "}" ;
match_arm = pattern , [ "if" , expression ] , "=>" , expression , [ "," ] ;
```

**match文の例**:
```orizon
// 基本的なパターンマッチング
match value {
    1 => println!("one"),
    2 => println!("two"),
    3 => println!("three"),
    _ => println!("something else"),
}

// 範囲パターン
match age {
    0..=12 => println!("child"),
    13..=19 => println!("teenager"),
    20..=59 => println!("adult"),
    60.. => println!("senior"),
}

// 構造体パターン
match point {
    Point { x: 0, y: 0 } => println!("origin"),
    Point { x: 0, y } => println!("on y-axis at {}", y),
    Point { x, y: 0 } => println!("on x-axis at {}", x),
    Point { x, y } => println!("({}, {})", x, y),
}

// ガード付きパターン
match value {
    x if x < 0 => println!("negative: {}", x),
    x if x > 0 => println!("positive: {}", x),
    _ => println!("zero"),
}

// Option型のパターンマッチング
match optional_value {
    Some(x) => println!("値: {}", x),
    None => println!("値なし"),
}

// Result型のパターンマッチング
match result {
    Ok(value) => println!("成功: {}", value),
    Err(error) => println!("エラー: {}", error),
}
```

---

## 構造体とenum

### 構造体定義
```ebnf
struct_declaration = "struct" , identifier , [ generic_parameters ] , 
                     ( struct_fields | tuple_struct_fields ) ,
                     [ where_clause ] , ";" ;

struct_fields = "{" , [ field_list ] , "}" ;
field_list = field , { "," , field } , [ "," ] ;
field = [ "pub" ] , identifier , ":" , type ;

tuple_struct_fields = "(" , [ type_list ] , ")" ;
```

**構造体の例**:
```orizon
// 基本的な構造体
struct Point {
    x: f64,
    y: f64,
}

// パブリックフィールド
struct Person {
    pub name: String,
    pub age: u32,
    private_id: u64,
}

// タプル構造体
struct Color(u8, u8, u8);
struct Wrapper(String);

// ユニット構造体
struct Marker;

// ジェネリック構造体
struct Container<T> {
    data: T,
    metadata: String,
}

// ライフタイム付き構造体
struct Reference<'a> {
    data: &'a str,
}
```

### 構造体の使用
```orizon
// 構造体インスタンスの作成
let point = Point { x: 3.0, y: 4.0 };

// フィールドアクセス
let x_coord = point.x;

// 可変構造体
let mut person = Person {
    name: "Alice".to_string(),
    age: 30,
    private_id: 12345,
};
person.age += 1;

// 構造体更新構文
let point2 = Point { x: 5.0, ..point };  // y: 4.0 を引き継ぐ

// タプル構造体の使用
let red = Color(255, 0, 0);
let red_value = red.0;  // 255

// パターンマッチング
match point {
    Point { x: 0.0, y: 0.0 } => println!("原点"),
    Point { x, y } => println!("({}, {})", x, y),
}
```

### enum定義
```ebnf
enum_declaration = "enum" , identifier , [ generic_parameters ] , "{" , 
                   [ enum_variant_list ] , "}" , [ where_clause ] ;

enum_variant_list = enum_variant , { "," , enum_variant } , [ "," ] ;
enum_variant = identifier , [ enum_variant_data ] ;
enum_variant_data = "(" , [ type_list ] , ")" | "{" , [ field_list ] , "}" ;
```

**enumの例**:
```orizon
// 基本的なenum
enum Direction {
    North,
    South,
    East,
    West,
}

// データ付きenum
enum Message {
    Quit,
    Move { x: i32, y: i32 },
    Write(String),
    ChangeColor(u8, u8, u8),
}

// ジェネリックenum
enum Option<T> {
    Some(T),
    None,
}

enum Result<T, E> {
    Ok(T),
    Err(E),
}

// 複雑なenum
enum Expression {
    Number(f64),
    Variable(String),
    Add(Box<Expression>, Box<Expression>),
    Multiply(Box<Expression>, Box<Expression>),
}
```

### enumの使用
```orizon
// enumの使用
let direction = Direction::North;

let msg = Message::Move { x: 10, y: 20 };

// パターンマッチング
match msg {
    Message::Quit => println!("終了"),
    Message::Move { x, y } => println!("移動: ({}, {})", x, y),
    Message::Write(text) => println!("書き込み: {}", text),
    Message::ChangeColor(r, g, b) => println!("色変更: RGB({}, {}, {})", r, g, b),
}

// if letパターン
if let Message::Write(text) = msg {
    println!("メッセージ: {}", text);
}
```

---

## トレイト（trait）

### トレイト定義
```ebnf
trait_declaration = "trait" , identifier , [ generic_parameters ] , 
                    [ ":" , trait_bounds ] , [ where_clause ] , "{" , 
                    { trait_item } , "}" ;

trait_item = trait_function | trait_type | trait_const ;
trait_function = "func" , identifier , function_signature , [ block_statement ] ;
trait_type = "type" , identifier , [ ":" , trait_bounds ] , ";" ;
trait_const = "const" , identifier , ":" , type , [ "=" , expression ] , ";" ;
```

**トレイトの例**:
```orizon
// 基本的なトレイト
trait Display {
    func fmt(&self) -> String;
}

// デフォルト実装付きトレイト
trait Drawable {
    func draw(&self);
    
    func draw_twice(&self) {  // デフォルト実装
        self.draw();
        self.draw();
    }
}

// ジェネリックトレイト
trait Iterator<T> {
    func next(&mut self) -> Option<T>;
    
    func collect<C: FromIterator<T>>(&mut self) -> C {
        C::from_iter(self)
    }
}

// 関連型付きトレイト
trait IntoIterator {
    type Item;
    type IntoIter: Iterator<Item = Self::Item>;
    
    func into_iter(self) -> Self::IntoIter;
}

// 関連定数付きトレイト
trait Number {
    const ZERO: Self;
    const ONE: Self;
    
    func is_zero(&self) -> bool;
}
```

### トレイト実装
```ebnf
impl_declaration = "impl" , [ generic_parameters ] , 
                   [ trait_reference , "for" ] , type , 
                   [ where_clause ] , "{" , { impl_item } , "}" ;
```

**トレイト実装の例**:
```orizon
// 基本的なトレイト実装
impl Display for Point {
    func fmt(&self) -> String {
        format!("({}, {})", self.x, self.y)
    }
}

// ジェネリック実装
impl<T: Display> Display for Vec<T> {
    func fmt(&self) -> String {
        let items: Vec<String> = self.iter()
            .map(|item| item.fmt())
            .collect();
        format!("[{}]", items.join(", "))
    }
}

// 条件付き実装
impl<T> Clone for Point<T> 
where 
    T: Clone 
{
    func clone(&self) -> Self {
        Point {
            x: self.x.clone(),
            y: self.y.clone(),
        }
    }
}

// 固有実装（トレイトなし）
impl Point {
    func new(x: f64, y: f64) -> Self {
        Point { x, y }
    }
    
    func distance(&self, other: &Point) -> f64 {
        ((self.x - other.x).powi(2) + (self.y - other.y).powi(2)).sqrt()
    }
}
```

---

## ジェネリクス

### ジェネリック型パラメータ
```ebnf
generic_parameters = "<" , generic_param_list , ">" ;
generic_param_list = generic_param , { "," , generic_param } , [ "," ] ;
generic_param = type_param | lifetime_param | const_param ;
type_param = identifier , [ ":" , trait_bounds ] , [ "=" , type ] ;
lifetime_param = "'" , identifier ;
const_param = "const" , identifier , ":" , type , [ "=" , expression ] ;
```

**ジェネリクスの例**:
```orizon
// ジェネリック関数
func swap<T>(a: &mut T, b: &mut T) {
    let temp = std::mem::replace(a, std::mem::replace(b, temp));
}

// 制約付きジェネリクス
func print_if_displayable<T: Display>(value: T) {
    println!("{}", value.fmt());
}

// 複数の制約
func process<T>(value: T) 
where 
    T: Clone + Display + Send + Sync 
{
    let cloned = value.clone();
    println!("{}", cloned.fmt());
}

// ライフタイムパラメータ
func longest<'a>(x: &'a str, y: &'a str) -> &'a str {
    if x.len() > y.len() { x } else { y }
}

// 定数ジェネリクス
struct Array<T, const N: usize> {
    data: [T; N],
}

impl<T, const N: usize> Array<T, N> {
    func new() -> Self {
        Array { data: [T::default(); N] }
    }
}
```

### Where句
```ebnf
where_clause = "where" , where_predicate_list ;
where_predicate_list = where_predicate , { "," , where_predicate } , [ "," ] ;
where_predicate = type , ":" , trait_bounds |
                  lifetime , ":" , lifetime_bounds ;
```

**Where句の例**:
```orizon
// 複雑な制約をWhere句で表現
func complex_function<T, U, V>(a: T, b: U) -> V
where
    T: Clone + Display,
    U: Into<String> + Debug,
    V: From<T> + Default,
{
    // 実装
}

// ライフタイム制約
func reference_function<'a, 'b, T>(x: &'a T, y: &'b T) -> &'a T
where
    'b: 'a,  // 'b は 'a より長生きする
    T: Display,
{
    x
}
```

---

## モジュールシステム

### モジュール定義
```ebnf
module_declaration = "mod" , identifier , ( ";" | "{" , { item } , "}" ) ;
```

**モジュールの例**:
```orizon
// インラインモジュール
mod math {
    pub func add(a: i32, b: i32) -> i32 {
        a + b
    }
    
    func internal_helper() {
        // プライベート関数
    }
    
    pub mod advanced {
        pub func complex_calculation() -> f64 {
            // 複雑な計算
        }
    }
}

// 外部ファイルモジュール
mod network;  // network.oriz または network/mod.oriz を読み込み
```

### use文
```ebnf
use_declaration = "use" , use_tree , ";" ;
use_tree = use_path | use_path , "::" , "{" , use_tree_list , "}" | 
           use_path , "::" , "*" | use_path , "as" , identifier ;
```

**use文の例**:
```orizon
// 単純なインポート
use std::collections::HashMap;

// 複数インポート
use std::collections::{HashMap, HashSet, BTreeMap};

// 全てインポート
use std::prelude::*;

// エイリアス
use std::collections::HashMap as Map;

// 相対インポート
use super::parent_module::function;
use self::child_module::Type;

// 再エクスポート
pub use internal_module::PublicType;
```

### 可視性
```orizon
// パブリック
pub struct PublicStruct {
    pub public_field: i32,
    private_field: String,
}

// クレート内パブリック
pub(crate) func crate_visible_function() {}

// スーパーモジュール内パブリック
pub(super) func parent_visible_function() {}

// 指定モジュール内パブリック
pub(in crate::specific::module) func module_visible_function() {}
```

---

## 非同期プログラミング

### async/await構文
```ebnf
async_function = "async" , function_declaration ;
await_expression = expression , "." , "await" ;
```

**非同期プログラミングの例**:
```orizon
// 非同期関数
async func fetch_url(url: &str) -> Result<String, Error> {
    let response = http_client.get(url).await?;
    let text = response.text().await?;
    Ok(text)
}

// 非同期ブロック
let future = async {
    let data = fetch_url("https://api.example.com").await?;
    process_data(data)
};

// 非同期ストリーム
async func process_stream(mut stream: impl AsyncStream<Item = Data>) {
    while let Some(item) = stream.next().await {
        process_item(item).await;
    }
}

// 並行実行
async func concurrent_requests() -> Vec<String> {
    let futures = vec![
        fetch_url("https://api1.example.com"),
        fetch_url("https://api2.example.com"),
        fetch_url("https://api3.example.com"),
    ];
    
    futures::future::join_all(futures).await
}
```

---

## マクロシステム

### マクロ定義
```ebnf
macro_declaration = "macro" , identifier , "(" , macro_params , ")" , "{" , macro_body , "}" ;
```

**マクロの例**:
```orizon
// 宣言的マクロ
macro_rules! vec {
    () => {
        Vec::new()
    };
    ($elem:expr; $n:expr) => {
        vec![].resize($n, $elem)
    };
    ($($x:expr),+ $(,)?) => {
        {
            let mut temp_vec = Vec::new();
            $(
                temp_vec.push($x);
            )+
            temp_vec
        }
    };
}

// 手続き的マクロ
#[derive(Debug, Clone)]
struct Person {
    name: String,
    age: u32,
}

// カスタム derive マクロ
#[derive(Serialize, Deserialize)]
struct ApiResponse {
    status: String,
    data: Value,
}
```

---

## エラーハンドリング

### Result型とOption型
```orizon
// Result型の使用
func divide(a: f64, b: f64) -> Result<f64, String> {
    if b == 0.0 {
        Err("ゼロ除算エラー".to_string())
    } else {
        Ok(a / b)
    }
}

// ?演算子
func complex_operation() -> Result<i32, Error> {
    let file = File::open("data.txt")?;
    let content = file.read_to_string()?;
    let number = content.trim().parse::<i32>()?;
    Ok(number * 2)
}

// Option型の使用
func find_user(id: u32) -> Option<User> {
    database.users.get(&id).cloned()
}

// パターンマッチングでのエラーハンドリング
match divide(10.0, 3.0) {
    Ok(result) => println!("結果: {}", result),
    Err(error) => println!("エラー: {}", error),
}

// if letパターン
if let Some(user) = find_user(123) {
    println!("ユーザー: {}", user.name);
}
```

### パニックハンドリング
```orizon
// パニック発生
func dangerous_operation() {
    panic!("重大なエラーが発生しました");
}

// アサーション
func validate_input(value: i32) {
    assert!(value > 0, "値は正数である必要があります");
    assert_eq!(value % 2, 0, "値は偶数である必要があります");
}

// パニックキャッチ
func safe_operation() -> Result<i32, Box<dyn std::error::Error>> {
    let result = std::panic::catch_unwind(|| {
        dangerous_operation();
        42
    });
    
    match result {
        Ok(value) => Ok(value),
        Err(_) => Err("パニックが発生しました".into()),
    }
}
```

---

## 高度な構文機能

### パターンマッチング詳細
```orizon
// 複雑なパターンマッチング
match value {
    // リテラルパターン
    42 => println!("答え"),
    
    // 変数パターン
    x => println!("値: {}", x),
    
    // ワイルドカードパターン
    _ => println!("その他"),
}

// 構造化パターン
match person {
    Person { name, age: 0..=17 } => println!("未成年: {}", name),
    Person { name, age: 18..=64 } => println!("成人: {}", name),
    Person { name, age: 65.. } => println!("高齢者: {}", name),
}

// ガードパターン
match point {
    Point { x, y } if x == y => println!("対角線上の点"),
    Point { x: 0, y } => println!("Y軸上の点: {}", y),
    Point { x, y: 0 } => println!("X軸上の点: {}", x),
    Point { x, y } => println!("一般的な点: ({}, {})", x, y),
}

// 参照パターン
match &option_value {
    Some(ref inner) => println!("値: {}", inner),
    None => println!("なし"),
}
```

### 属性（Attributes）
```orizon
// 関数属性
#[inline]
#[must_use]
func important_calculation() -> i32 {
    // 重要な計算
}

// 構造体属性
#[derive(Debug, Clone, PartialEq)]
#[repr(C)]
struct CCompatibleStruct {
    field1: i32,
    field2: f64,
}

// 条件付きコンパイル
#[cfg(target_os = "windows")]
func windows_specific_function() {
    // Windows固有の実装
}

#[cfg(feature = "advanced")]
mod advanced_features {
    // 高度な機能（feature flagで制御）
}
```

このガイドにより、Orizonプログラミング言語の完全な構文と文法を理解し、効果的なプログラムを作成することができます。
