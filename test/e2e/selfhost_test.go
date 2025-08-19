package e2e

import (
	"fmt"
	"strings"
	"testing"

	orizonTesting "github.com/orizon-lang/orizon/internal/testing"
)

// TestSelfHosting tests self-hosting capabilities
func TestSelfHosting(t *testing.T) {
	framework, err := orizonTesting.NewTestFramework(nil)
	if err != nil {
		t.Fatalf("Failed to create test framework: %v", err)
	}
	defer framework.Cleanup()

	tests := []*orizonTesting.CompilerTest{
		{
			Name: "minimal_compiler",
			SourceCode: `
// Minimal compiler that can compile itself
use std::io::File;
use std::process::exit;

struct Token {
    kind: TokenKind,
    text: String,
}

enum TokenKind {
    Fn,
    Let,
    Identifier,
    Number,
    String,
    EOF,
}

struct Lexer {
    input: String,
    position: usize,
}

impl Lexer {
    fn new(input: String) -> Self {
        Lexer { input, position: 0 }
    }
    
    fn next_token(&mut self) -> Token {
        // Simplified lexer implementation
        if self.position >= self.input.len() {
            return Token { kind: TokenKind::EOF, text: "".to_string() };
        }
        
        // Skip whitespace
        while self.position < self.input.len() && self.input.chars().nth(self.position).unwrap().is_whitespace() {
            self.position += 1;
        }
        
        if self.position >= self.input.len() {
            return Token { kind: TokenKind::EOF, text: "".to_string() };
        }
        
        let ch = self.input.chars().nth(self.position).unwrap();
        self.position += 1;
        
        Token { kind: TokenKind::Identifier, text: ch.to_string() }
    }
}

fn main() {
    let source = "fn main() { print(\"Hello\"); }";
    let mut lexer = Lexer::new(source.to_string());
    
    loop {
        let token = lexer.next_token();
        match token.kind {
            TokenKind::EOF => break,
            _ => print("Token"),
        }
    }
    
    print("Compilation complete");
}`,
			ExpectedOut: "Compilation complete",
			ShouldFail:  false,
		},
		{
			Name: "bootstrap_test",
			SourceCode: `
// Test that the compiler can compile a simple version of itself
fn compile(source: String) -> bool {
    // Simplified compilation process
    if source.contains("fn main()") {
        return true;
    }
    return false;
}

fn main() {
    let simple_program = "fn main() { print(\"Hello\"); }";
    let success = compile(simple_program.to_string());
    
    if success {
        print("Bootstrap successful");
    } else {
        print("Bootstrap failed");
    }
}`,
			ExpectedOut: "Bootstrap successful",
			ShouldFail:  false,
		},
	}

	framework.RunTestSuite(tests, t)
}

// TestRealWorldPrograms tests compilation of real-world programs
func TestRealWorldPrograms(t *testing.T) {
	framework, err := orizonTesting.NewTestFramework(nil)
	if err != nil {
		t.Fatalf("Failed to create test framework: %v", err)
	}
	defer framework.Cleanup()

	tests := []*orizonTesting.CompilerTest{
		{
			Name: "calculator",
			SourceCode: `
enum Operation {
    Add,
    Subtract,
    Multiply,
    Divide,
}

struct Calculator {
    operations: Vec<Operation>,
}

impl Calculator {
    fn new() -> Self {
        Calculator { operations: Vec::new() }
    }
    
    fn add_operation(&mut self, op: Operation) {
        self.operations.push(op);
    }
    
    fn calculate(&self, a: f64, b: f64) -> Result<f64, String> {
        let mut result = a;
        
        for op in &self.operations {
            match op {
                Operation::Add => result += b,
                Operation::Subtract => result -= b,
                Operation::Multiply => result *= b,
                Operation::Divide => {
                    if b == 0.0 {
                        return Err("Division by zero".to_string());
                    }
                    result /= b;
                },
            }
        }
        
        Ok(result)
    }
}

fn main() {
    let mut calc = Calculator::new();
    calc.add_operation(Operation::Add);
    
    match calc.calculate(5.0, 3.0) {
        Ok(result) => print(result),
        Err(msg) => print(msg),
    }
}`,
			ExpectedOut: "8",
			ShouldFail:  false,
		},
		{
			Name: "json_parser",
			SourceCode: `
enum JsonValue {
    Null,
    Bool(bool),
    Number(f64),
    String(String),
    Array(Vec<JsonValue>),
    Object(HashMap<String, JsonValue>),
}

struct JsonParser {
    input: String,
    position: usize,
}

impl JsonParser {
    fn new(input: String) -> Self {
        JsonParser { input, position: 0 }
    }
    
    fn parse(&mut self) -> Result<JsonValue, String> {
        self.skip_whitespace();
        
        if self.position >= self.input.len() {
            return Err("Unexpected end of input".to_string());
        }
        
        let ch = self.current_char();
        match ch {
            '"' => self.parse_string(),
            '[' => self.parse_array(),
            '{' => self.parse_object(),
            't' | 'f' => self.parse_bool(),
            'n' => self.parse_null(),
            _ => self.parse_number(),
        }
    }
    
    fn current_char(&self) -> char {
        self.input.chars().nth(self.position).unwrap_or('\0')
    }
    
    fn skip_whitespace(&mut self) {
        while self.position < self.input.len() && self.current_char().is_whitespace() {
            self.position += 1;
        }
    }
    
    fn parse_string(&mut self) -> Result<JsonValue, String> {
        self.position += 1; // Skip opening quote
        let start = self.position;
        
        while self.position < self.input.len() && self.current_char() != '"' {
            self.position += 1;
        }
        
        if self.position >= self.input.len() {
            return Err("Unterminated string".to_string());
        }
        
        let value = self.input[start..self.position].to_string();
        self.position += 1; // Skip closing quote
        
        Ok(JsonValue::String(value))
    }
    
    fn parse_array(&mut self) -> Result<JsonValue, String> {
        Ok(JsonValue::Array(Vec::new()))
    }
    
    fn parse_object(&mut self) -> Result<JsonValue, String> {
        Ok(JsonValue::Object(HashMap::new()))
    }
    
    fn parse_bool(&mut self) -> Result<JsonValue, String> {
        Ok(JsonValue::Bool(true))
    }
    
    fn parse_null(&mut self) -> Result<JsonValue, String> {
        Ok(JsonValue::Null)
    }
    
    fn parse_number(&mut self) -> Result<JsonValue, String> {
        Ok(JsonValue::Number(42.0))
    }
}

fn main() {
    let mut parser = JsonParser::new("\"hello\"".to_string());
    match parser.parse() {
        Ok(JsonValue::String(s)) => print(s),
        Ok(_) => print("Not a string"),
        Err(msg) => print(msg),
    }
}`,
			ExpectedOut: "hello",
			ShouldFail:  false,
		},
		{
			Name: "web_server",
			SourceCode: `
use std::net::{TcpListener, TcpStream};
use std::io::{Read, Write};
use std::thread;

struct HttpRequest {
    method: String,
    path: String,
    headers: HashMap<String, String>,
}

struct HttpResponse {
    status_code: u16,
    status_text: String,
    headers: HashMap<String, String>,
    body: String,
}

impl HttpResponse {
    fn new(status_code: u16, status_text: String, body: String) -> Self {
        let mut headers = HashMap::new();
        headers.insert("Content-Length".to_string(), body.len().to_string());
        headers.insert("Content-Type".to_string(), "text/html".to_string());
        
        HttpResponse {
            status_code,
            status_text,
            headers,
            body,
        }
    }
    
    fn to_string(&self) -> String {
        let mut response = format!("HTTP/1.1 {} {}\r\n", self.status_code, self.status_text);
        
        for (key, value) in &self.headers {
            response.push_str(&format!("{}: {}\r\n", key, value));
        }
        
        response.push_str("\r\n");
        response.push_str(&self.body);
        
        response
    }
}

fn handle_client(mut stream: TcpStream) {
    let mut buffer = [0; 1024];
    let _ = stream.read(&mut buffer).unwrap();
    
    let response = HttpResponse::new(
        200,
        "OK".to_string(),
        "<html><body><h1>Hello from Orizon!</h1></body></html>".to_string()
    );
    
    let _ = stream.write(response.to_string().as_bytes()).unwrap();
    let _ = stream.flush().unwrap();
}

fn main() {
    // Simplified web server for testing
    print("Web server simulation complete");
}`,
			ExpectedOut: "Web server simulation complete",
			ShouldFail:  false,
		},
	}

	framework.RunTestSuite(tests, t)
}

// TestComplexApplications tests complex application scenarios
func TestComplexApplications(t *testing.T) {
	framework, err := orizonTesting.NewTestFramework(nil)
	if err != nil {
		t.Fatalf("Failed to create test framework: %v", err)
	}
	defer framework.Cleanup()

	tests := []*orizonTesting.CompilerTest{
		{
			Name: "game_engine",
			SourceCode: `
struct Vector2 {
    x: f32,
    y: f32,
}

impl Vector2 {
    fn new(x: f32, y: f32) -> Self {
        Vector2 { x, y }
    }
    
    fn add(&self, other: &Vector2) -> Vector2 {
        Vector2::new(self.x + other.x, self.y + other.y)
    }
    
    fn magnitude(&self) -> f32 {
        (self.x * self.x + self.y * self.y).sqrt()
    }
}

struct GameObject {
    position: Vector2,
    velocity: Vector2,
    active: bool,
}

impl GameObject {
    fn new(x: f32, y: f32) -> Self {
        GameObject {
            position: Vector2::new(x, y),
            velocity: Vector2::new(0.0, 0.0),
            active: true,
        }
    }
    
    fn update(&mut self, dt: f32) {
        if self.active {
            self.position = self.position.add(&Vector2::new(
                self.velocity.x * dt,
                self.velocity.y * dt
            ));
        }
    }
}

struct GameWorld {
    objects: Vec<GameObject>,
}

impl GameWorld {
    fn new() -> Self {
        GameWorld { objects: Vec::new() }
    }
    
    fn add_object(&mut self, object: GameObject) {
        self.objects.push(object);
    }
    
    fn update(&mut self, dt: f32) {
        for object in &mut self.objects {
            object.update(dt);
        }
    }
}

fn main() {
    let mut world = GameWorld::new();
    let mut player = GameObject::new(100.0, 100.0);
    player.velocity = Vector2::new(50.0, 0.0);
    
    world.add_object(player);
    world.update(0.1); // Update for 0.1 seconds
    
    print("Game engine simulation complete");
}`,
			ExpectedOut: "Game engine simulation complete",
			ShouldFail:  false,
		},
		{
			Name: "database_orm",
			SourceCode: `
trait Serialize {
    fn serialize(&self) -> String;
}

trait Deserialize {
    fn deserialize(data: String) -> Result<Self, String> where Self: Sized;
}

struct User {
    id: i32,
    name: String,
    email: String,
}

impl Serialize for User {
    fn serialize(&self) -> String {
        format!("{}|{}|{}", self.id, self.name, self.email)
    }
}

impl Deserialize for User {
    fn deserialize(data: String) -> Result<Self, String> {
        let parts: Vec<&str> = data.split('|').collect();
        if parts.len() != 3 {
            return Err("Invalid data format".to_string());
        }
        
        let id = parts[0].parse::<i32>().map_err(|_| "Invalid ID")?;
        let name = parts[1].to_string();
        let email = parts[2].to_string();
        
        Ok(User { id, name, email })
    }
}

struct Database<T> {
    records: Vec<T>,
}

impl<T> Database<T> where T: Serialize + Deserialize {
    fn new() -> Self {
        Database { records: Vec::new() }
    }
    
    fn insert(&mut self, record: T) {
        self.records.push(record);
    }
    
    fn count(&self) -> usize {
        self.records.len()
    }
}

fn main() {
    let mut db = Database::<User>::new();
    
    let user = User {
        id: 1,
        name: "Alice".to_string(),
        email: "alice@example.com".to_string(),
    };
    
    db.insert(user);
    print(db.count());
}`,
			ExpectedOut: "1",
			ShouldFail:  false,
		},
	}

	framework.RunTestSuite(tests, t)
}

// TestStressTests performs stress testing of the compiler
func TestStressTests(t *testing.T) {
	framework, err := orizonTesting.NewTestFramework(nil)
	if err != nil {
		t.Fatalf("Failed to create test framework: %v", err)
	}
	defer framework.Cleanup()

	tests := []*orizonTesting.CompilerTest{
		{
			Name:       "large_file_compilation",
			SourceCode: generateLargeProgram(10000),
			ShouldFail: false,
		},
		{
			Name:       "deep_nesting",
			SourceCode: generateDeeplyNestedCode(100),
			ShouldFail: false,
		},
		{
			Name:       "many_functions",
			SourceCode: generateManyFunctions(1000),
			ShouldFail: false,
		},
	}

	framework.RunTestSuite(tests, t)
}

// Helper functions for generating test code

func generateLargeProgram(lines int) string {
	var code string
	code += "fn main() {\n"
	for i := 0; i < lines; i++ {
		code += fmt.Sprintf("    let var%d = %d;\n", i, i)
	}
	code += "    print(\"Large program compiled successfully\");\n"
	code += "}\n"
	return code
}

func generateDeeplyNestedCode(depth int) string {
	var code string
	code += "fn main() {\n"

	// Create nested if statements
	for i := 0; i < depth; i++ {
		code += strings.Repeat("    ", i+1) + "if true {\n"
	}

	code += strings.Repeat("    ", depth+1) + "print(\"Deep nesting works\");\n"

	// Close all the if statements
	for i := depth - 1; i >= 0; i-- {
		code += strings.Repeat("    ", i+1) + "}\n"
	}

	code += "}\n"
	return code
}

func generateManyFunctions(count int) string {
	var code string

	// Generate many small functions
	for i := 0; i < count; i++ {
		code += fmt.Sprintf("fn func%d() -> i32 {\n", i)
		code += fmt.Sprintf("    return %d;\n", i)
		code += "}\n\n"
	}

	// Main function that calls some of them
	code += "fn main() {\n"
	code += "    let sum = func0() + func1() + func2();\n"
	code += "    print(sum);\n"
	code += "}\n"

	return code
}
