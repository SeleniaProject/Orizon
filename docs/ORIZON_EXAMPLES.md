# Orizon Programming Language - Practical Examples

This document provides comprehensive examples that demonstrate the Orizon language syntax and standard library usage, linking to the test files created in the repository.

## Table of Contents

1. [Basic Language Features](#basic-language-features)
2. [Data Structures and Collections](#data-structures-and-collections)
3. [Control Flow and Loops](#control-flow-and-loops)
4. [Functions and Methods](#functions-and-methods)
5. [Object-Oriented Programming](#object-oriented-programming)
6. [Concurrency and Parallelism](#concurrency-and-parallelism)
7. [Error Handling](#error-handling)
8. [Standard Library Integration](#standard-library-integration)

---

## Basic Language Features

### Variable Declaration and Types

*Reference: [simple_test.oriz](../test_simple.oriz)*

```orizon
// Basic variable declarations
let name: string = "Alice"
let age: int = 25
let salary: float = 50000.0
let isActive: bool = true

// Type inference
let city = "New York"        // inferred as string
let temperature = 23.5       // inferred as float
let count = 42              // inferred as int

// Constants
const PI: float = 3.14159
const MAX_USERS: int = 1000

// Nullable types
let middleName: string? = null
let optionalAge: int? = 30

// Arrays and slices
let numbers: [int] = [1, 2, 3, 4, 5]
let names: []string = ["Alice", "Bob", "Charlie"]

// Maps
let scores: map[string]int = {
    "Alice": 95,
    "Bob": 87,
    "Charlie": 92
}

// Tuples
let coordinate: (float, float) = (10.5, 20.3)
let person: (string, int, bool) = ("Alice", 25, true)
```

### String Operations

```orizon
// String concatenation
let firstName = "John"
let lastName = "Doe"
let fullName = firstName + " " + lastName

// String interpolation
let greeting = "Hello, {firstName}! You are {age} years old."

// String methods
let text = "  Hello, World!  "
let trimmed = text.trim()
let upper = text.toUpper()
let lower = text.toLower()
let length = text.length()

// String slicing
let substring = text[7:12]  // "World"
let prefix = text[:5]       // "  Hel"
let suffix = text[7:]       // "World!  "

// String formatting
let formatted = string.format("Name: {}, Age: {}, Salary: {:.2f}", name, age, salary)
```

---

## Data Structures and Collections

### Arrays and Dynamic Arrays

*Reference: [test_array.oriz](../test_array.oriz)*

```orizon
import "stdlib/collections"

// Fixed-size arrays
let fixedArray: [5]int = [1, 2, 3, 4, 5]

// Dynamic arrays (slices)
let dynamicArray: []int = []
dynamicArray.append(10)
dynamicArray.append(20)
dynamicArray.append(30)

// Array operations
let first = dynamicArray[0]
let last = dynamicArray[dynamicArray.length() - 1]
let slice = dynamicArray[1:3]  // [20, 30]

// Array methods
dynamicArray.insert(1, 15)     // Insert 15 at index 1
dynamicArray.remove(2)         // Remove element at index 2
let index = dynamicArray.indexOf(20)  // Find index of element

// Array iteration
for (index, value) in dynamicArray {
    println("Index: {}, Value: {}", index, value)
}

// Array comprehension
let squares = [x * x for x in 1..10]
let evenNumbers = [x for x in 1..20 if x % 2 == 0]
```

### Maps and Dictionaries

```orizon
// Map declaration and initialization
let userAges: map[string]int = {
    "Alice": 25,
    "Bob": 30,
    "Charlie": 35
}

// Adding and updating entries
userAges["Diana"] = 28
userAges["Alice"] = 26  // Update existing

// Checking existence
if userAges.hasKey("Alice") {
    println("Alice's age: {}", userAges["Alice"])
}

// Safe access with default
let unknownAge = userAges.get("Eve", 0)  // Returns 0 if key doesn't exist

// Removing entries
userAges.remove("Bob")

// Map iteration
for (name, age) in userAges {
    println("{} is {} years old", name, age)
}

// Getting keys and values
let names = userAges.keys()
let ages = userAges.values()
```

### Advanced Collections

```orizon
import "stdlib/collections"

// Set operations
let uniqueNumbers = collections.NewSet[int]()
uniqueNumbers.add(1)
uniqueNumbers.add(2)
uniqueNumbers.add(1)  // Duplicate, will be ignored

let hasTwo = uniqueNumbers.contains(2)
let size = uniqueNumbers.size()

// Set operations
let setA = collections.SetFromArray([1, 2, 3, 4])
let setB = collections.SetFromArray([3, 4, 5, 6])

let union = setA.union(setB)          // [1, 2, 3, 4, 5, 6]
let intersection = setA.intersect(setB)  // [3, 4]
let difference = setA.difference(setB)   // [1, 2]

// Priority Queue
let pq = collections.NewPriorityQueue[Task](func(a, b Task) bool {
    return a.priority > b.priority  // Higher priority first
})

pq.push(Task{name: "Important", priority: 10})
pq.push(Task{name: "Normal", priority: 5})
pq.push(Task{name: "Urgent", priority: 15})

while !pq.empty() {
    task := pq.pop()
    println("Processing: {}", task.name)
}

// Stack and Queue
let stack = collections.NewStack[string]()
stack.push("first")
stack.push("second")
let top = stack.pop()  // "second"

let queue = collections.NewQueue[int]()
queue.enqueue(1)
queue.enqueue(2)
let front = queue.dequeue()  // 1
```

---

## Control Flow and Loops

### Conditional Statements

*Reference: [test_if.oriz](../test_if.oriz)*

```orizon
// Basic if-else
let score = 85

if score >= 90 {
    println("Grade: A")
} else if score >= 80 {
    println("Grade: B")
} else if score >= 70 {
    println("Grade: C")
} else {
    println("Grade: F")
}

// Ternary operator
let status = age >= 18 ? "Adult" : "Minor"

// Switch statement
let day = "Monday"
switch day {
case "Monday", "Tuesday", "Wednesday", "Thursday", "Friday":
    println("Weekday")
case "Saturday", "Sunday":
    println("Weekend")
default:
    println("Invalid day")
}

// Switch with expressions
let category = switch score {
case 90..100: "Excellent"
case 80..89:  "Good"
case 70..79:  "Fair"
case 60..69:  "Poor"
default:      "Fail"
}

// Pattern matching
let result = match value {
case Some(x) if x > 0: "Positive: {x}"
case Some(x) if x < 0: "Negative: {x}"
case Some(0):          "Zero"
case None:             "No value"
}
```

### Loop Constructs

*Reference: [test_for_in.oriz](../test_for_in.oriz), [test_while.oriz](../test_while.oriz)*

```orizon
// For loops
for i in 0..10 {
    println("Number: {}", i)
}

// For loop with step
for i in 0..20 step 2 {
    println("Even number: {}", i)
}

// Reverse iteration
for i in 10..0 step -1 {
    println("Countdown: {}", i)
}

// Array iteration
let fruits = ["apple", "banana", "orange"]
for fruit in fruits {
    println("Fruit: {}", fruit)
}

// Index and value iteration
for (index, fruit) in fruits {
    println("Index {}: {}", index, fruit)
}

// Map iteration
for (key, value) in userAges {
    println("{}: {}", key, value)
}

// While loops
let count = 0
while count < 5 {
    println("Count: {}", count)
    count++
}

// Do-while loops
let input: string
do {
    input = readLine("Enter 'quit' to exit: ")
} while input != "quit"

// Loop control
for i in 0..100 {
    if i % 2 == 0 {
        continue  // Skip even numbers
    }
    
    if i > 50 {
        break    // Exit loop when i > 50
    }
    
    println("Odd number: {}", i)
}

// Labeled breaks and continues
outer: for i in 0..10 {
    for j in 0..10 {
        if i * j > 25 {
            break outer  // Break from outer loop
        }
        println("i: {}, j: {}, product: {}", i, j, i * j)
    }
}
```

---

## Functions and Methods

### Function Declaration and Usage

*Reference: [test_function.oriz](../test_function.oriz)*

```orizon
// Basic function
func greet(name: string) {
    println("Hello, {}!", name)
}

// Function with return value
func add(a: int, b: int) -> int {
    return a + b
}

// Function with multiple return values
func divmod(a: int, b: int) -> (int, int) {
    return (a / b, a % b)
}

// Function with optional parameters
func createUser(name: string, age: int = 18, isActive: bool = true) -> User {
    return User{
        name: name,
        age: age,
        isActive: isActive,
        createdAt: time.now()
    }
}

// Variadic functions
func sum(numbers: ...int) -> int {
    let total = 0
    for num in numbers {
        total += num
    }
    return total
}

let result = sum(1, 2, 3, 4, 5)  // 15

// Higher-order functions
func applyOperation(a: int, b: int, op: func(int, int) -> int) -> int {
    return op(a, b)
}

let multiply = func(x: int, y: int) -> int { return x * y }
let product = applyOperation(5, 3, multiply)  // 15

// Lambda expressions
let square = |x: int| -> int { return x * x }
let isEven = |x: int| -> bool { return x % 2 == 0 }

// Function composition
func compose[T, U, V](f: func(U) -> V, g: func(T) -> U) -> func(T) -> V {
    return |x: T| -> V { return f(g(x)) }
}

let addOne = |x: int| -> int { return x + 1 }
let double = |x: int| -> int { return x * 2 }
let addOneThenDouble = compose(double, addOne)

println(addOneThenDouble(5))  // (5 + 1) * 2 = 12
```

### Generic Functions

```orizon
// Generic function
func swap[T](a: T, b: T) -> (T, T) {
    return (b, a)
}

let (x, y) = swap(10, 20)
let (str1, str2) = swap("hello", "world")

// Generic function with constraints
func max[T: Comparable](a: T, b: T) -> T {
    return a > b ? a : b
}

// Generic function with multiple type parameters
func zip[T, U](list1: []T, list2: []U) -> [](T, U) {
    let result: [](T, U) = []
    let minLen = min(list1.length(), list2.length())
    
    for i in 0..minLen {
        result.append((list1[i], list2[i]))
    }
    
    return result
}

let zipped = zip([1, 2, 3], ["a", "b", "c"])  // [(1, "a"), (2, "b"), (3, "c")]
```

---

## Object-Oriented Programming

### Structs and Methods

*Reference: [test_struct.oriz](../test_struct.oriz)*

```orizon
// Struct definition
struct Person {
    name: string
    age: int
    email: string
    private ssn: string  // Private field
}

// Constructor function
func NewPerson(name: string, age: int, email: string, ssn: string) -> Person {
    return Person{
        name: name,
        age: age,
        email: email,
        ssn: ssn
    }
}

// Methods
impl Person {
    // Instance method
    func (self) getName() -> string {
        return self.name
    }
    
    // Mutating method
    func (mut self) setAge(newAge: int) {
        if newAge >= 0 {
            self.age = newAge
        }
    }
    
    // Method with parameters
    func (self) isOlderThan(other: Person) -> bool {
        return self.age > other.age
    }
    
    // Static method
    static func createMinor(name: string, email: string) -> Person {
        return Person{
            name: name,
            age: 0,
            email: email,
            ssn: ""
        }
    }
    
    // Property getter
    func (self) fullInfo() -> string {
        return "{} ({} years old) - {}" % (self.name, self.age, self.email)
    }
}

// Usage
let person = NewPerson("Alice", 25, "alice@example.com", "123-45-6789")
println(person.getName())
person.setAge(26)

let minor = Person.createMinor("Bob", "bob@example.com")
```

### Inheritance and Interfaces

```orizon
// Interface definition
interface Drawable {
    func draw()
    func getArea() -> float
}

interface Movable {
    func move(dx: float, dy: float)
    func getPosition() -> (float, float)
}

// Struct implementing interfaces
struct Circle {
    x: float
    y: float
    radius: float
}

impl Drawable for Circle {
    func (self) draw() {
        println("Drawing circle at ({}, {}) with radius {}", self.x, self.y, self.radius)
    }
    
    func (self) getArea() -> float {
        return PI * self.radius * self.radius
    }
}

impl Movable for Circle {
    func (mut self) move(dx: float, dy: float) {
        self.x += dx
        self.y += dy
    }
    
    func (self) getPosition() -> (float, float) {
        return (self.x, self.y)
    }
}

// Multiple interfaces
struct Rectangle {
    x: float
    y: float
    width: float
    height: float
}

impl Drawable for Rectangle {
    func (self) draw() {
        println("Drawing rectangle at ({}, {}) with size {}x{}", 
                self.x, self.y, self.width, self.height)
    }
    
    func (self) getArea() -> float {
        return self.width * self.height
    }
}

impl Movable for Rectangle {
    func (mut self) move(dx: float, dy: float) {
        self.x += dx
        self.y += dy
    }
    
    func (self) getPosition() -> (float, float) {
        return (self.x, self.y)
    }
}

// Polymorphism
func drawShapes(shapes: []Drawable) {
    for shape in shapes {
        shape.draw()
        println("Area: {}", shape.getArea())
    }
}

let shapes: []Drawable = [
    Circle{x: 0, y: 0, radius: 5},
    Rectangle{x: 10, y: 10, width: 8, height: 6}
]

drawShapes(shapes)
```

### Composition and Embedding

```orizon
// Composition
struct Engine {
    horsepower: int
    fuelType: string
    
    func (self) start() {
        println("Engine starting... {} HP, {}", self.horsepower, self.fuelType)
    }
    
    func (self) stop() {
        println("Engine stopping...")
    }
}

struct Car {
    make: string
    model: string
    year: int
    engine: Engine  // Composition
    
    func (self) startCar() {
        println("Starting {} {} {}", self.year, self.make, self.model)
        self.engine.start()
    }
    
    func (self) stopCar() {
        self.engine.stop()
        println("Car stopped")
    }
}

// Embedding (anonymous fields)
struct Vehicle {
    Engine  // Embedded struct
    make: string
    model: string
}

impl Vehicle {
    func (self) honk() {
        println("Beep beep!")
    }
}

// Usage
let car = Car{
    make: "Toyota",
    model: "Camry", 
    year: 2023,
    engine: Engine{horsepower: 200, fuelType: "Gasoline"}
}

car.startCar()

let vehicle = Vehicle{
    Engine: Engine{horsepower: 150, fuelType: "Electric"},
    make: "Tesla",
    model: "Model 3"
}

vehicle.start()  // Can call Engine methods directly
vehicle.honk()
```

---

## Concurrency and Parallelism

### Goroutines and Channels

*Reference: [test_async.oriz](../test_async.oriz)*

```orizon
import "stdlib/concurrent"

// Basic goroutine
func worker(id: int) {
    for i in 0..5 {
        println("Worker {}: {}", id, i)
        concurrent.sleep(1*time.Second)
    }
}

// Start goroutines
go worker(1)
go worker(2)
go worker(3)

// Channel communication
let ch = make(chan int, 10)  // Buffered channel

// Send data to channel
go func() {
    for i in 0..5 {
        ch <- i
        concurrent.sleep(500*time.Millisecond)
    }
    close(ch)
}()

// Receive data from channel
for value := range ch {
    println("Received: {}", value)
}

// Select statement for multiple channels
let ch1 = make(chan string)
let ch2 = make(chan int)
let done = make(chan bool)

go func() {
    concurrent.sleep(2*time.Second)
    ch1 <- "Hello"
}()

go func() {
    concurrent.sleep(1*time.Second)
    ch2 <- 42
}()

go func() {
    concurrent.sleep(3*time.Second)
    done <- true
}()

for {
    select {
    case msg := <-ch1:
        println("String received: {}", msg)
    case num := <-ch2:
        println("Number received: {}", num)
    case <-done:
        println("Done!")
        return
    case <-time.After(5*time.Second):
        println("Timeout!")
        return
    }
}
```

### Worker Pools and Pipeline Patterns

```orizon
import "stdlib/concurrent"

// Worker pool pattern
func workerPool() {
    const numWorkers = 3
    jobs := make(chan Job, 100)
    results := make(chan Result, 100)
    
    // Start workers
    for w := 1; w <= numWorkers; w++ {
        go worker(w, jobs, results)
    }
    
    // Send jobs
    go func() {
        for j := 1; j <= 10; j++ {
            jobs <- Job{ID: j, Data: "job-{}" % j}
        }
        close(jobs)
    }()
    
    // Collect results
    for r := 1; r <= 10; r++ {
        result := <-results
        println("Result: {}", result)
    }
}

func worker(id: int, jobs <-chan Job, results chan<- Result) {
    for job := range jobs {
        println("Worker {} processing job {}", id, job.ID)
        
        // Simulate work
        concurrent.sleep(1*time.Second)
        
        results <- Result{
            JobID: job.ID,
            Data: "processed-{}" % job.Data,
            WorkerID: id
        }
    }
}

// Pipeline pattern
func pipeline() {
    // Stage 1: Generate numbers
    numbers := make(chan int)
    go func() {
        for i in 1..100 {
            numbers <- i
        }
        close(numbers)
    }()
    
    // Stage 2: Square numbers
    squares := make(chan int)
    go func() {
        for num := range numbers {
            squares <- num * num
        }
        close(squares)
    }()
    
    // Stage 3: Filter even squares
    evenSquares := make(chan int)
    go func() {
        for square := range squares {
            if square % 2 == 0 {
                evenSquares <- square
            }
        }
        close(evenSquares)
    }()
    
    // Output
    for even := range evenSquares {
        println("Even square: {}", even)
    }
}
```

### Synchronization Primitives

```orizon
import "stdlib/concurrent"

// Mutex for protecting shared data
let mut counter = 0
let counterMutex = concurrent.NewMutex()

func incrementCounter() {
    counterMutex.lock()
    defer counterMutex.unlock()
    
    counter++
}

// WaitGroup for waiting for goroutines
let wg = concurrent.NewWaitGroup()

for i in 0..10 {
    wg.add(1)
    go func(id: int) {
        defer wg.done()
        
        // Do work
        incrementCounter()
        println("Goroutine {} finished", id)
    }(i)
}

wg.wait()
println("Final counter value: {}", counter)

// Once for one-time initialization
let once = concurrent.NewOnce()
let mut initialized = false

func expensiveInitialization() {
    once.do(func() {
        println("Performing expensive initialization...")
        concurrent.sleep(2*time.Second)
        initialized = true
    })
}

// Condition variables
let cond = concurrent.NewCond(&counterMutex)
let mut ready = false

// Producer
go func() {
    concurrent.sleep(2*time.Second)
    counterMutex.lock()
    ready = true
    cond.broadcast()  // Wake up all waiting goroutines
    counterMutex.unlock()
}()

// Consumers
for i in 0..3 {
    go func(id: int) {
        counterMutex.lock()
        for !ready {
            cond.wait()  // Wait until ready becomes true
        }
        println("Consumer {} proceeding", id)
        counterMutex.unlock()
    }(i)
}
```

---

## Error Handling

### Result Types and Error Propagation

*Reference: [test_error.oriz](../test_error.oriz)*

```orizon
// Result type for operations that can fail
type Result[T, E] = Ok(T) | Err(E)

// Function that can fail
func divide(a: float, b: float) -> Result[float, string] {
    if b == 0.0 {
        return Err("Division by zero")
    }
    return Ok(a / b)
}

// Error handling with pattern matching
let result = divide(10.0, 2.0)
match result {
case Ok(value):
    println("Result: {}", value)
case Err(error):
    println("Error: {}", error)
}

// Error propagation with ? operator
func calculateAverage(numbers: []float) -> Result[float, string] {
    if numbers.length() == 0 {
        return Err("Cannot calculate average of empty array")
    }
    
    let sum = 0.0
    for num in numbers {
        sum += num
    }
    
    return divide(sum, numbers.length() as float)?  // Propagate error if division fails
}

// Chaining operations
func processData(data: []string) -> Result[[]int, string] {
    let numbers: []int = []
    
    for item in data {
        let num = parseInt(item)?  // Propagate parse error
        numbers.append(num)
    }
    
    return Ok(numbers)
}

// Option type for nullable values
type Option[T] = Some(T) | None

func findUser(id: int) -> Option[User] {
    // Simulate database lookup
    if id == 1 {
        return Some(User{id: 1, name: "Alice"})
    }
    return None
}

// Working with Option
let user = findUser(1)
match user {
case Some(u):
    println("Found user: {}", u.name)
case None:
    println("User not found")
}

// Option methods
let username = findUser(1).map(|u| u.name).unwrapOr("Unknown")
```

### Custom Error Types

```orizon
// Custom error interface
interface Error {
    func message() -> string
    func code() -> int
}

// Specific error types
struct ValidationError {
    field: string
    reason: string
}

impl Error for ValidationError {
    func (self) message() -> string {
        return "Validation failed for field '{}': {}" % (self.field, self.reason)
    }
    
    func (self) code() -> int {
        return 400
    }
}

struct DatabaseError {
    operation: string
    underlying: string
}

impl Error for DatabaseError {
    func (self) message() -> string {
        return "Database error during {}: {}" % (self.operation, self.underlying)
    }
    
    func (self) code() -> int {
        return 500
    }
}

// Function using custom errors
func createUser(data: UserData) -> Result[User, Error] {
    // Validation
    if data.email.length() == 0 {
        return Err(ValidationError{
            field: "email",
            reason: "Email is required"
        })
    }
    
    if !isValidEmail(data.email) {
        return Err(ValidationError{
            field: "email", 
            reason: "Invalid email format"
        })
    }
    
    // Database operation
    user := database.save(data) catch |dbErr| {
        return Err(DatabaseError{
            operation: "create_user",
            underlying: dbErr.message()
        })
    }
    
    return Ok(user)
}

// Error handling with different error types
let result = createUser(userData)
match result {
case Ok(user):
    println("User created: {}", user.id)
case Err(err):
    match err {
    case ValidationError(validErr):
        println("Validation error: {}", validErr.message())
        // Return 400 to client
    case DatabaseError(dbErr):
        println("Database error: {}", dbErr.message())
        // Return 500 to client
    default:
        println("Unknown error: {}", err.message())
    }
}
```

### Panic and Recovery

```orizon
// Panic for unrecoverable errors
func validateInput(value: int) {
    if value < 0 {
        panic("Negative values not allowed")
    }
}

// Defer and recover
func safeDivision(a: int, b: int) -> Option[int] {
    defer func() {
        if err := recover(); err != nil {
            println("Recovered from panic: {}", err)
        }
    }()
    
    if b == 0 {
        panic("Division by zero")
    }
    
    return Some(a / b)
}

// Using try-catch for exceptions
func processFile(filename: string) -> Result[string, Error] {
    return try {
        let file = io.open(filename)?
        defer file.close()
        
        let content = file.readAll()?
        Ok(content)
    } catch IOError(err) {
        Err(FileError{filename: filename, cause: err})
    } catch PermissionError(err) {
        Err(SecurityError{operation: "read_file", details: err})
    }
}
```

---

## Standard Library Integration

### File I/O and System Operations

```orizon
import "stdlib/io"
import "stdlib/os"

// File operations
func fileOperations() {
    // Write to file
    let content = "Hello, World!\nThis is a test file."
    io.writeFile("test.txt", content.bytes()) catch |err| {
        println("Failed to write file: {}", err)
        return
    }
    
    // Read from file
    let data = io.readFile("test.txt") catch |err| {
        println("Failed to read file: {}", err)
        return
    }
    
    let text = string.fromBytes(data)
    println("File content: {}", text)
    
    // File information
    let info = io.stat("test.txt") catch |err| {
        println("Failed to get file info: {}", err)
        return
    }
    
    println("File size: {} bytes", info.size)
    println("Modified: {}", info.modTime)
    
    // Directory operations
    os.mkdir("testdir") catch |err| {
        println("Failed to create directory: {}", err)
    }
    
    let entries = io.readDir("testdir") catch |err| {
        println("Failed to read directory: {}", err)
        return
    }
    
    for entry in entries {
        println("Entry: {}, IsDir: {}", entry.name, entry.isDir)
    }
}
```

### Network Programming

```orizon
import "stdlib/network"
import "stdlib/web"

// HTTP client
func httpClient() {
    let client = network.NewHTTPClient(network.HTTPClientConfig{
        timeout: 30*time.Second,
        maxRetries: 3,
        userAgent: "Orizon/1.0",
    })
    
    // GET request
    let response = client.get("https://api.example.com/users") catch |err| {
        println("HTTP request failed: {}", err)
        return
    }
    
    defer response.close()
    
    if response.statusCode == 200 {
        let body = response.readAll() catch |err| {
            println("Failed to read response: {}", err)
            return
        }
        
        println("Response: {}", string.fromBytes(body))
    }
    
    // POST request with JSON
    let userData = {
        "name": "Alice",
        "email": "alice@example.com"
    }
    
    let jsonData = json.marshal(userData) catch |err| {
        println("Failed to marshal JSON: {}", err)
        return
    }
    
    let postResponse = client.post("https://api.example.com/users", 
                                   jsonData, 
                                   "application/json") catch |err| {
        println("POST request failed: {}", err)
        return
    }
    
    println("User created: {}", postResponse.statusCode == 201)
}

// TCP server
func tcpServer() {
    let listener = network.listen("tcp", ":8080") catch |err| {
        println("Failed to start server: {}", err)
        return
    }
    
    defer listener.close()
    println("Server listening on :8080")
    
    for {
        let conn = listener.accept() catch |err| {
            println("Failed to accept connection: {}", err)
            continue
        }
        
        // Handle connection in goroutine
        go handleConnection(conn)
    }
}

func handleConnection(conn: network.Connection) {
    defer conn.close()
    
    let buffer = make([]byte, 1024)
    for {
        let n = conn.read(buffer) catch |err| {
            println("Connection read error: {}", err)
            break
        }
        
        if n == 0 {
            break  // Connection closed
        }
        
        let message = string.fromBytes(buffer[:n])
        println("Received: {}", message)
        
        // Echo back
        conn.write(buffer[:n]) catch |err| {
            println("Connection write error: {}", err)
            break
        }
    }
}
```

### JSON and Data Serialization

```orizon
import "stdlib/json"
import "stdlib/yaml"

struct Config {
    appName: string
    port: int
    debug: bool
    database: DatabaseConfig
}

struct DatabaseConfig {
    host: string
    port: int
    username: string
    password: string
}

// JSON serialization
func jsonExample() {
    let config = Config{
        appName: "MyApp",
        port: 8080,
        debug: true,
        database: DatabaseConfig{
            host: "localhost",
            port: 5432,
            username: "admin",
            password: "secret"
        }
    }
    
    // Serialize to JSON
    let jsonData = json.marshal(config) catch |err| {
        println("JSON marshal error: {}", err)
        return
    }
    
    println("JSON: {}", string.fromBytes(jsonData))
    
    // Deserialize from JSON
    let parsedConfig: Config = json.unmarshal(jsonData, Config) catch |err| {
        println("JSON unmarshal error: {}", err)
        return
    }
    
    println("Parsed app name: {}", parsedConfig.appName)
}

// YAML configuration
func yamlExample() {
    let yamlContent = `
app_name: MyApp
port: 8080
debug: true
database:
  host: localhost
  port: 5432
  username: admin
  password: secret
`
    
    let config: Config = yaml.unmarshal(yamlContent.bytes(), Config) catch |err| {
        println("YAML unmarshal error: {}", err)
        return
    }
    
    println("Loaded config: {}", config.appName)
    
    // Serialize to YAML
    let yamlData = yaml.marshal(config) catch |err| {
        println("YAML marshal error: {}", err)
        return
    }
    
    println("YAML output:\n{}", string.fromBytes(yamlData))
}
```

---

*This comprehensive example collection demonstrates the Orizon programming language features and standard library integration. Each example is designed to be practical and demonstrates real-world usage patterns.*
