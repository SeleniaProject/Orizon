# Orizon Standard Library - Feature Guides

## Table of Contents

1. [Machine Learning Development Guide](#machine-learning-development-guide)
2. [High-Performance Networking Guide](#high-performance-networking-guide)
3. [Database Integration Guide](#database-integration-guide)
4. [Web Application Development](#web-application-development)
5. [Graphics and Multimedia](#graphics-and-multimedia)
6. [Cryptography and Security](#cryptography-and-security)
7. [Cloud and Distributed Systems](#cloud-and-distributed-systems)
8. [Performance Optimization](#performance-optimization)

---

## Machine Learning Development Guide

### Getting Started with Deep Learning

The Orizon ML package provides a complete framework for building neural networks, from simple feedforward networks to complex transformer architectures.

#### Building Your First Neural Network

```orizon
import "stdlib/ml"

func buildClassifier() {
    // Create a neural network for image classification
    network := ml.NewNeuralNetwork()
    
    // Input layer (28x28 image = 784 features)
    network.AddLayer(ml.InputLayer{neurons: 784})
    
    // Hidden layers
    network.AddLayer(ml.DenseLayer{
        neurons: 128,
        activation: ml.ReLU,
        dropout: 0.2,
    })
    
    network.AddLayer(ml.DenseLayer{
        neurons: 64,
        activation: ml.ReLU,
        dropout: 0.3,
    })
    
    // Output layer (10 classes)
    network.AddLayer(ml.OutputLayer{
        neurons: 10,
        activation: ml.Softmax,
    })
    
    // Compile the model
    network.Compile(ml.CompileOptions{
        optimizer: ml.Adam{
            learningRate: 0.001,
            beta1: 0.9,
            beta2: 0.999,
        },
        loss: ml.CategoricalCrossentropy,
        metrics: [ml.Accuracy, ml.Precision, ml.Recall],
    })
    
    return network
}
```

#### Computer Vision Pipeline

```orizon
func buildImageClassificationPipeline() {
    // Load and preprocess data
    dataset := ml.LoadImageDataset("path/to/dataset")
    
    // Data augmentation
    augmenter := ml.NewDataAugmenter()
    augmenter.AddTransform(ml.RandomRotation{degrees: 15})
    augmenter.AddTransform(ml.RandomHorizontalFlip{probability: 0.5})
    augmenter.AddTransform(ml.RandomZoom{range: 0.1})
    
    // Create CNN architecture
    model := ml.NewSequentialModel()
    
    // Convolutional layers
    model.Add(ml.Conv2D{
        filters: 32,
        kernelSize: (3, 3),
        activation: ml.ReLU,
        inputShape: (224, 224, 3),
    })
    model.Add(ml.MaxPooling2D{poolSize: (2, 2)})
    
    model.Add(ml.Conv2D{
        filters: 64,
        kernelSize: (3, 3),
        activation: ml.ReLU,
    })
    model.Add(ml.MaxPooling2D{poolSize: (2, 2)})
    
    model.Add(ml.Conv2D{
        filters: 128,
        kernelSize: (3, 3),
        activation: ml.ReLU,
    })
    model.Add(ml.GlobalAveragePooling2D{})
    
    // Classification head
    model.Add(ml.Dense{neurons: 512, activation: ml.ReLU})
    model.Add(ml.Dropout{rate: 0.5})
    model.Add(ml.Dense{neurons: dataset.NumClasses, activation: ml.Softmax})
    
    // Training with callbacks
    callbacks := []ml.Callback{
        ml.EarlyStopping{
            monitor: "val_accuracy",
            patience: 10,
            restoreBestWeights: true,
        },
        ml.ReduceLROnPlateau{
            monitor: "val_loss",
            factor: 0.5,
            patience: 5,
        },
        ml.ModelCheckpoint{
            filepath: "best_model.h5",
            saveOnlyBest: true,
        },
    }
    
    // Train the model
    history := model.Fit(ml.FitOptions{
        trainData: dataset.Train,
        validationData: dataset.Validation,
        epochs: 100,
        batchSize: 32,
        callbacks: callbacks,
    })
    
    // Evaluate on test set
    evaluation := model.Evaluate(dataset.Test)
    println("Test Accuracy: {:.4f}", evaluation.Accuracy)
}
```

#### Natural Language Processing

```orizon
func buildTextClassifier() {
    // Text preprocessing
    tokenizer := ml.NewTokenizer(ml.TokenizerOptions{
        vocabularySize: 10000,
        oovToken: "<UNK>",
        padToken: "<PAD>",
    })
    
    // Load and tokenize text data
    texts := loadTextData("reviews.txt")
    tokenized := tokenizer.TextsToSequences(texts)
    padded := ml.PadSequences(tokenized, maxLength: 256)
    
    // Build LSTM model for sentiment analysis
    model := ml.NewSequentialModel()
    
    model.Add(ml.Embedding{
        vocabularySize: tokenizer.VocabularySize,
        embeddingDim: 128,
        inputLength: 256,
    })
    
    model.Add(ml.LSTM{
        units: 64,
        dropout: 0.5,
        recurrentDropout: 0.5,
    })
    
    model.Add(ml.Dense{neurons: 32, activation: ml.ReLU})
    model.Add(ml.Dense{neurons: 1, activation: ml.Sigmoid})
    
    // Compile and train
    model.Compile(ml.CompileOptions{
        optimizer: ml.RMSprop{learningRate: 0.001},
        loss: ml.BinaryCrossentropy,
        metrics: [ml.Accuracy],
    })
    
    model.Fit(ml.FitOptions{
        x: padded,
        y: labels,
        epochs: 20,
        batchSize: 64,
        validationSplit: 0.2,
    })
}
```

#### Transfer Learning

```orizon
func transferLearningExample() {
    // Load pre-trained model
    baseModel := ml.LoadPretrainedModel("ResNet50", ml.PretrainedOptions{
        weights: "imagenet",
        includeTop: false,
        inputShape: (224, 224, 3),
    })
    
    // Freeze base model layers
    baseModel.Trainable = false
    
    // Add custom classification head
    model := ml.NewFunctionalModel()
    x := baseModel.Output
    x = ml.GlobalAveragePooling2D()(x)
    x = ml.Dense{neurons: 1024, activation: ml.ReLU}(x)
    x = ml.Dropout{rate: 0.5}(x)
    predictions := ml.Dense{neurons: numClasses, activation: ml.Softmax}(x)
    
    model.Compile(ml.CompileOptions{
        optimizer: ml.Adam{learningRate: 0.0001},
        loss: ml.CategoricalCrossentropy,
        metrics: [ml.Accuracy],
    })
    
    // Fine-tuning phase
    model.Fit(trainData, epochs: 10)
    
    // Unfreeze some layers for fine-tuning
    for i := len(baseModel.Layers) - 20; i < len(baseModel.Layers); i++ {
        baseModel.Layers[i].Trainable = true
    }
    
    model.Compile(ml.CompileOptions{
        optimizer: ml.Adam{learningRate: 0.00001}, // Lower learning rate
        loss: ml.CategoricalCrossentropy,
        metrics: [ml.Accuracy],
    })
    
    model.Fit(trainData, epochs: 10)
}
```

---

## High-Performance Networking Guide

### Zero-Copy Packet Processing

Orizon's networking package is designed for maximum performance with zero-copy operations and kernel bypass capabilities.

#### High-Speed Packet Capture

```orizon
import "stdlib/network"

func packetCapture() {
    // Create high-performance network interface
    interface := network.NewInterface("eth0", network.InterfaceOptions{
        mode: network.ZeroCopyMode,
        ringBuffer: network.RingBufferConfig{
            rxSize: 8192,
            txSize: 8192,
            frameSize: 2048,
        },
        polling: true,
        promisc: true,
    })
    
    // Set up packet filters
    filter := network.NewBPFFilter()
    filter.AddRule("tcp port 80 or tcp port 443")
    interface.SetFilter(filter)
    
    // Packet processing pipeline
    processor := network.NewPacketProcessor(network.ProcessorConfig{
        workers: runtime.NumCPU(),
        bufferSize: 1024,
        timeout: 1*time.Millisecond,
    })
    
    // Process packets in parallel
    for {
        batch := interface.ReceiveBatch(64) // Receive up to 64 packets
        
        for _, packet := range batch {
            // Zero-copy packet analysis
            analyzePacket(packet)
            
            // Forward if needed (still zero-copy)
            if shouldForward(packet) {
                interface.TransmitZeroCopy(packet)
            }
        }
    }
}

func analyzePacket(packet *network.Packet) {
    // Parse headers without copying data
    eth := packet.EthernetHeader()
    
    switch eth.EtherType {
    case network.EtherTypeIPv4:
        ip := packet.IPv4Header()
        switch ip.Protocol {
        case network.ProtocolTCP:
            tcp := packet.TCPHeader()
            // Analyze TCP packet
            analyzeTCP(ip, tcp, packet.Payload())
        case network.ProtocolUDP:
            udp := packet.UDPHeader()
            // Analyze UDP packet
            analyzeUDP(ip, udp, packet.Payload())
        }
    case network.EtherTypeIPv6:
        ip6 := packet.IPv6Header()
        // Analyze IPv6 packet
    }
}
```

#### Software-Defined Networking

```orizon
func setupSDNController() {
    // Create SDN controller
    controller := network.NewSDNController(network.ControllerConfig{
        listenPort: 6653, // OpenFlow port
        version: network.OpenFlow13,
    })
    
    // Network topology discovery
    topology := network.NewTopologyManager()
    topology.EnableLLDP(true)
    topology.EnableSTP(true)
    
    controller.SetTopologyManager(topology)
    
    // Define network policies
    policy := network.NewNetworkPolicy()
    
    // Micro-segmentation rules
    policy.AddRule(network.FlowRule{
        priority: 1000,
        match: network.FlowMatch{
            inPort: 1,
            ethType: network.EtherTypeIPv4,
            ipSrc: "192.168.1.0/24",
            ipDst: "10.0.0.0/8",
        },
        actions: []network.FlowAction{
            {type: network.ActionDrop},
        },
    })
    
    // Load balancing rule
    policy.AddRule(network.FlowRule{
        priority: 800,
        match: network.FlowMatch{
            ethType: network.EtherTypeIPv4,
            ipDst: "192.168.1.100", // VIP
            ipProto: network.ProtocolTCP,
            tcpDst: 80,
        },
        actions: []network.FlowAction{
            {type: network.ActionSetField, field: "ipv4_dst", value: selectBackend()},
            {type: network.ActionOutput, port: getBackendPort()},
        },
    })
    
    controller.SetPolicy(policy)
    
    // Handle switch connections
    controller.OnSwitchConnect = func(sw *network.Switch) {
        println("Switch connected: {}", sw.DPID)
        
        // Install default flows
        installDefaultFlows(sw)
        
        // Enable flow monitoring
        sw.EnableFlowStats(true)
        sw.EnablePortStats(true)
    }
    
    controller.Start()
}
```

#### Network Virtualization

```orizon
func setupNetworkVirtualization() {
    // Create VXLAN overlay network
    vxlan := network.NewVXLAN(network.VXLANConfig{
        vni: 100,
        mcastGroup: "239.1.1.1",
        port: 4789,
        interface: "eth0",
    })
    
    // Bridge configuration
    bridge := network.NewBridge("br-vxlan")
    bridge.AddInterface(vxlan.Interface())
    
    // Virtual machine network isolation
    vmNet := network.NewVMNetwork(network.VMNetworkConfig{
        bridge: bridge,
        vlanMode: network.VLANTrunk,
        security: network.SecurityConfig{
            antiSpoofing: true,
            firewallRules: []network.FirewallRule{
                {
                    direction: network.Ingress,
                    protocol: network.ProtocolTCP,
                    port: 22,
                    action: network.Allow,
                },
                {
                    direction: network.Egress,
                    action: network.Allow,
                },
            },
        },
    })
    
    // Network namespace isolation
    netns := network.NewNetworkNamespace("container1")
    netns.CreateVethPair("veth0", "veth1")
    netns.MoveInterface("veth1")
    netns.SetupRouting("10.1.1.0/24")
    
    // Quality of Service
    qos := network.NewQoSManager()
    qos.AddClass(network.QoSClass{
        name: "high_priority",
        rate: "100Mbps",
        ceil: "1Gbps",
        priority: 1,
    })
    
    qos.AddFilter(network.QoSFilter{
        protocol: network.ProtocolTCP,
        dstPort: 80,
        class: "high_priority",
    })
    
    qos.Apply("eth0")
}
```

#### Network Monitoring and Analytics

```orizon
func networkMonitoring() {
    // Create monitoring system
    monitor := network.NewNetworkMonitor(network.MonitorConfig{
        interfaces: []string{"eth0", "eth1"},
        metrics: []string{"throughput", "latency", "packet_loss", "jitter"},
        interval: 1*time.Second,
    })
    
    // Flow monitoring
    flowMonitor := network.NewFlowMonitor()
    flowMonitor.EnableNetFlow(true)
    flowMonitor.EnableSFlow(true)
    
    // Anomaly detection
    anomalyDetector := network.NewAnomalyDetector(network.AnomalyConfig{
        algorithms: []string{"isolation_forest", "one_class_svm"},
        windowSize: 300, // 5 minutes
        threshold: 0.05,
    })
    
    // Real-time analytics
    analytics := network.NewNetworkAnalytics()
    
    monitor.OnMetric = func(metric network.Metric) {
        // Store metric
        analytics.StoreMetric(metric)
        
        // Check for anomalies
        if anomalyDetector.IsAnomaly(metric) {
            alert := network.Alert{
                type: network.AnomalyAlert,
                severity: network.Warning,
                message: fmt.Sprintf("Anomaly detected: %s", metric.Name),
                timestamp: time.Now(),
            }
            handleAlert(alert)
        }
        
        // Update dashboard
        updateDashboard(metric)
    }
    
    monitor.Start()
}
```

---

## Database Integration Guide

### Multi-Database Architecture

Orizon supports seamless integration with multiple database types through a unified interface.

#### Unified Database Interface

```orizon
import "stdlib/database"

func setupDatabaseCluster() {
    // Create connection pools for different database types
    pgPool := database.NewConnectionPool(database.PoolConfig{
        driver: database.PostgreSQL,
        dsn: "postgres://user:pass@pg-cluster:5432/mydb",
        maxConnections: 100,
        minConnections: 10,
        maxIdleTime: 30*time.Minute,
        healthCheck: true,
        retryPolicy: database.ExponentialBackoff{
            maxRetries: 3,
            baseDelay: 1*time.Second,
        },
    })
    
    redisPool := database.NewConnectionPool(database.PoolConfig{
        driver: database.Redis,
        dsn: "redis://redis-cluster:6379",
        maxConnections: 50,
        commandTimeout: 5*time.Second,
        clusterMode: true,
    })
    
    mongoPool := database.NewConnectionPool(database.PoolConfig{
        driver: database.MongoDB,
        dsn: "mongodb://mongo-replica:27017",
        maxConnections: 75,
        readPreference: database.ReadPreferenceSecondary,
        writeConcern: database.WriteConcernMajority,
    })
    
    // Create database manager with multiple backends
    dbManager := database.NewDatabaseManager()
    dbManager.AddPool("primary", pgPool)
    dbManager.AddPool("cache", redisPool)
    dbManager.AddPool("documents", mongoPool)
    
    return dbManager
}
```

#### Database Sharding

```orizon
func setupSharding() {
    sharding := database.NewShardingManager(database.ShardingConfig{
        shardKey: "user_id",
        algorithm: database.ConsistentHashing,
        replication: 3,
        consistency: database.EventualConsistency,
    })
    
    // Add shards
    for i := 0; i < 8; i++ {
        shard := database.NewDatabaseManager(database.Config{
            driver: database.PostgreSQL,
            host: fmt.Sprintf("shard-%d.example.com", i),
            port: 5432,
            database: "shard",
            ssl: true,
        })
        
        sharding.AddShard(fmt.Sprintf("shard-%d", i), shard, database.ShardConfig{
            weight: 1.0,
            region: fmt.Sprintf("region-%d", i%3),
            readOnly: false,
        })
    }
    
    // Cross-shard transactions
    tx := sharding.BeginDistributedTransaction()
    defer tx.Rollback()
    
    // Operations on multiple shards
    tx.Execute("shard-1", "UPDATE users SET balance = balance - ? WHERE user_id = ?", 100, 12345)
    tx.Execute("shard-3", "UPDATE users SET balance = balance + ? WHERE user_id = ?", 100, 67890)
    
    // Two-phase commit
    if err := tx.Commit(); err != nil {
        log.Printf("Transaction failed: %v", err)
        return
    }
}
```

#### Advanced Query Optimization

```orizon
func queryOptimization() {
    db := setupDatabaseCluster()
    
    // Query builder with optimization hints
    query := database.NewQueryBuilder("users")
    query.Select("id", "name", "email")
    query.Where("age > ?", 18)
    query.Where("status = ?", "active")
    query.OrderBy("created_at DESC")
    query.Limit(100)
    
    // Add optimization hints
    query.UseIndex("idx_age_status")
    query.SetCacheStrategy(database.CacheStrategy{
        enabled: true,
        ttl: 5*time.Minute,
        tags: []string{"users", "active"},
    })
    
    // Parallel query execution
    queries := []database.Query{
        query.Clone().Where("region = ?", "us-east"),
        query.Clone().Where("region = ?", "us-west"),
        query.Clone().Where("region = ?", "eu-west"),
    }
    
    results := database.ExecuteParallel(db, queries)
    
    // Merge results
    var allUsers []User
    for _, result := range results {
        users := []User{}
        result.Scan(&users)
        allUsers = append(allUsers, users...)
    }
    
    // Query analysis and optimization suggestions
    analyzer := database.NewQueryAnalyzer()
    plan := analyzer.Explain(query)
    
    if plan.HasSlowOperation() {
        suggestions := analyzer.GetOptimizationSuggestions(plan)
        for _, suggestion := range suggestions {
            log.Printf("Optimization suggestion: %s", suggestion)
        }
    }
}
```

#### Real-Time Data Streaming

```orizon
func setupDataStreaming() {
    // Database change streams
    stream := database.NewChangeStream(database.StreamConfig{
        database: "myapp",
        collection: "users",
        operations: []database.OperationType{
            database.Insert,
            database.Update,
            database.Delete,
        },
        resumeAfter: nil, // Start from latest
    })
    
    // Event processing pipeline
    processor := database.NewEventProcessor()
    
    processor.AddHandler(database.Insert, func(event database.ChangeEvent) {
        // Handle new user registration
        user := event.FullDocument.(User)
        sendWelcomeEmail(user.Email)
        updateAnalytics("user_registered", user.ID)
    })
    
    processor.AddHandler(database.Update, func(event database.ChangeEvent) {
        // Handle user updates
        if event.UpdatedFields["email"] != nil {
            user := loadUser(event.DocumentKey.ID)
            sendEmailChangeNotification(user)
        }
    })
    
    // Stream processing with backpressure handling
    for event := range stream.Watch() {
        select {
        case processor.Events <- event:
            // Event queued successfully
        case <-time.After(1*time.Second):
            // Handle backpressure
            log.Printf("Event queue full, dropping event: %v", event.ID)
        }
    }
}
```

---

## Web Application Development

### Modern Web Framework

Orizon's web framework provides everything needed for building modern web applications and APIs.

#### RESTful API with Advanced Features

```orizon
import "stdlib/web"

func createWebServer() {
    server := web.NewServer(web.ServerConfig{
        port: 8080,
        readTimeout: 30*time.Second,
        writeTimeout: 30*time.Second,
        maxHeaderBytes: 1 << 20, // 1MB
        enableHTTP2: true,
        enableHTTP3: true,
    })
    
    // Middleware stack
    server.Use(web.LoggingMiddleware(web.LoggingConfig{
        format: web.LogFormatJSON,
        includeRequestBody: true,
        includeResponseBody: false,
    }))
    
    server.Use(web.CORSMiddleware(web.CORSConfig{
        allowOrigins: []string{"https://app.example.com"},
        allowMethods: []string{"GET", "POST", "PUT", "DELETE"},
        allowHeaders: []string{"Content-Type", "Authorization"},
        exposeHeaders: []string{"X-Total-Count"},
        allowCredentials: true,
        maxAge: 3600,
    }))
    
    server.Use(web.RateLimitMiddleware(web.RateLimitConfig{
        requests: 100,
        window: time.Minute,
        keyFunc: func(ctx *web.Context) string {
            return ctx.ClientIP()
        },
        onLimitReached: func(ctx *web.Context) {
            ctx.JSON(429, web.H{"error": "Rate limit exceeded"})
        },
    }))
    
    server.Use(web.CompressionMiddleware(web.CompressionConfig{
        algorithms: []web.CompressionAlgorithm{
            web.Brotli,
            web.Gzip,
            web.Deflate,
        },
        minSize: 1024,
        excludeExtensions: []string{".png", ".jpg", ".gif"},
    }))
    
    // API versioning
    v1 := server.Group("/api/v1")
    v1.Use(web.AuthMiddleware(authConfig))
    
    setupUserRoutes(v1)
    setupProductRoutes(v1)
    setupOrderRoutes(v1)
    
    // Health check endpoint
    server.GET("/health", func(ctx *web.Context) {
        health := checkSystemHealth()
        status := 200
        if !health.Healthy {
            status = 503
        }
        ctx.JSON(status, health)
    })
    
    // Metrics endpoint
    server.GET("/metrics", web.PrometheusHandler())
    
    server.Start()
}

func setupUserRoutes(group *web.RouterGroup) {
    users := group.Group("/users")
    
    // Advanced parameter validation
    users.POST("/", web.ValidateJSON(UserCreateRequest{}), func(ctx *web.Context) {
        var req UserCreateRequest
        if err := ctx.ShouldBindJSON(&req); err != nil {
            ctx.JSON(400, web.H{"error": err.Error()})
            return
        }
        
        // Business logic
        user, err := userService.Create(req)
        if err != nil {
            ctx.JSON(500, web.H{"error": err.Error()})
            return
        }
        
        ctx.JSON(201, user)
    })
    
    // Pagination and filtering
    users.GET("/", func(ctx *web.Context) {
        page := ctx.DefaultQuery("page", "1")
        limit := ctx.DefaultQuery("limit", "10")
        filter := ctx.Query("filter")
        sort := ctx.DefaultQuery("sort", "created_at")
        
        users, total, err := userService.List(UserListOptions{
            Page: parseInt(page),
            Limit: parseInt(limit),
            Filter: filter,
            Sort: sort,
        })
        
        if err != nil {
            ctx.JSON(500, web.H{"error": err.Error()})
            return
        }
        
        ctx.Header("X-Total-Count", strconv.Itoa(total))
        ctx.JSON(200, users)
    })
    
    // File upload with validation
    users.POST("/:id/avatar", func(ctx *web.Context) {
        file, err := ctx.FormFile("avatar")
        if err != nil {
            ctx.JSON(400, web.H{"error": "No file uploaded"})
            return
        }
        
        // Validate file
        if file.Size > 5*1024*1024 { // 5MB limit
            ctx.JSON(400, web.H{"error": "File too large"})
            return
        }
        
        if !isValidImageType(file.Header.Get("Content-Type")) {
            ctx.JSON(400, web.H{"error": "Invalid file type"})
            return
        }
        
        // Process and save file
        avatarURL, err := fileService.SaveAvatar(ctx.Param("id"), file)
        if err != nil {
            ctx.JSON(500, web.H{"error": err.Error()})
            return
        }
        
        ctx.JSON(200, web.H{"avatarURL": avatarURL})
    })
}
```

#### GraphQL Server

```orizon
func setupGraphQLServer() {
    schema := web.NewGraphQLSchema()
    
    // Define types
    userType := schema.AddType("User", web.GraphQLType{
        fields: map[string]web.GraphQLField{
            "id": {
                type: web.NonNull(web.ID),
                description: "Unique identifier for the user",
            },
            "username": {
                type: web.NonNull(web.String),
                description: "User's display name",
            },
            "email": {
                type: web.NonNull(web.String),
                description: "User's email address",
            },
            "posts": {
                type: web.ListOf("Post"),
                description: "Posts created by the user",
                resolve: func(user User, args map[string]interface{}) []Post {
                    return postService.GetByUserID(user.ID)
                },
            },
            "followers": {
                type: web.ListOf("User"),
                description: "Users following this user",
                resolve: func(user User, args map[string]interface{}) []User {
                    return userService.GetFollowers(user.ID)
                },
            },
        },
    })
    
    postType := schema.AddType("Post", web.GraphQLType{
        fields: map[string]web.GraphQLField{
            "id": {type: web.NonNull(web.ID)},
            "title": {type: web.NonNull(web.String)},
            "content": {type: web.String},
            "author": {
                type: userType,
                resolve: func(post Post, args map[string]interface{}) User {
                    return userService.GetByID(post.AuthorID)
                },
            },
            "comments": {
                type: web.ListOf("Comment"),
                args: map[string]web.GraphQLArgument{
                    "first": {type: web.Int, defaultValue: 10},
                    "after": {type: web.String},
                },
                resolve: func(post Post, args map[string]interface{}) []Comment {
                    return commentService.GetByPostID(post.ID, args)
                },
            },
        },
    })
    
    // Root query
    schema.SetQuery(web.GraphQLType{
        fields: map[string]web.GraphQLField{
            "user": {
                type: userType,
                args: map[string]web.GraphQLArgument{
                    "id": {type: web.NonNull(web.ID)},
                },
                resolve: func(root interface{}, args map[string]interface{}) User {
                    return userService.GetByID(args["id"].(string))
                },
            },
            "users": {
                type: web.ListOf(userType),
                args: map[string]web.GraphQLArgument{
                    "first": {type: web.Int, defaultValue: 10},
                    "after": {type: web.String},
                    "filter": {type: "UserFilter"},
                },
                resolve: func(root interface{}, args map[string]interface{}) []User {
                    return userService.List(args)
                },
            },
        },
    })
    
    // Mutations
    schema.SetMutation(web.GraphQLType{
        fields: map[string]web.GraphQLField{
            "createUser": {
                type: userType,
                args: map[string]web.GraphQLArgument{
                    "input": {type: web.NonNull("CreateUserInput")},
                },
                resolve: func(root interface{}, args map[string]interface{}) User {
                    input := args["input"].(CreateUserInput)
                    return userService.Create(input)
                },
            },
            "updateUser": {
                type: userType,
                args: map[string]web.GraphQLArgument{
                    "id": {type: web.NonNull(web.ID)},
                    "input": {type: web.NonNull("UpdateUserInput")},
                },
                resolve: func(root interface{}, args map[string]interface{}) User {
                    id := args["id"].(string)
                    input := args["input"].(UpdateUserInput)
                    return userService.Update(id, input)
                },
            },
        },
    })
    
    // Subscriptions for real-time updates
    schema.SetSubscription(web.GraphQLType{
        fields: map[string]web.GraphQLField{
            "userUpdated": {
                type: userType,
                args: map[string]web.GraphQLArgument{
                    "id": {type: web.NonNull(web.ID)},
                },
                subscribe: func(root interface{}, args map[string]interface{}) <-chan User {
                    userID := args["id"].(string)
                    return userService.SubscribeToUpdates(userID)
                },
            },
            "newPost": {
                type: postType,
                subscribe: func(root interface{}, args map[string]interface{}) <-chan Post {
                    return postService.SubscribeToNew()
                },
            },
        },
    })
    
    // GraphQL server with extensions
    server := web.NewGraphQLServer(schema, web.GraphQLServerConfig{
        playground: true,
        introspection: true,
        extensions: []web.GraphQLExtension{
            web.NewTracingExtension(),
            web.NewCachingExtension(),
            web.NewComplexityAnalysis(web.ComplexityConfig{
                maximumComplexity: 1000,
                scalarCost: 1,
                objectCost: 2,
                listFactor: 10,
            }),
        },
    })
    
    server.Start(":8080")
}
```

#### WebSocket Real-Time Features

```orizon
func setupWebSocketServer() {
    wsManager := web.NewWebSocketManager(web.WebSocketConfig{
        checkOrigin: func(r *http.Request) bool {
            origin := r.Header.Get("Origin")
            return isAllowedOrigin(origin)
        },
        bufferSize: 1024,
        readTimeout: 60*time.Second,
        writeTimeout: 10*time.Second,
        enableCompression: true,
    })
    
    // Chat room implementation
    chatRoom := web.NewRoom("general")
    
    http.HandleFunc("/ws/chat", func(w http.ResponseWriter, r *http.Request) {
        conn, err := wsManager.Upgrade(w, r)
        if err != nil {
            log.Printf("WebSocket upgrade failed: %v", err)
            return
        }
        
        // Authenticate user
        token := r.URL.Query().Get("token")
        user, err := authService.ValidateToken(token)
        if err != nil {
            conn.Close(web.CloseUnauthorized, "Authentication failed")
            return
        }
        
        // Join room
        client := chatRoom.Join(conn, user.ID)
        
        // Send welcome message
        client.Send(web.Message{
            Type: "welcome",
            Data: map[string]interface{}{
                "message": "Welcome to the chat!",
                "onlineUsers": chatRoom.GetOnlineUsers(),
            },
        })
        
        // Broadcast user joined
        chatRoom.Broadcast(web.Message{
            Type: "userJoined",
            Data: map[string]interface{}{
                "userID": user.ID,
                "username": user.Username,
            },
        }, client.ID)
        
        // Handle messages
        for {
            var message ChatMessage
            if err := client.ReadJSON(&message); err != nil {
                log.Printf("Read error: %v", err)
                break
            }
            
            // Validate and process message
            if len(message.Content) > 1000 {
                client.Send(web.Message{
                    Type: "error",
                    Data: map[string]interface{}{
                        "message": "Message too long",
                    },
                })
                continue
            }
            
            // Store message
            savedMessage := chatService.SaveMessage(ChatMessage{
                RoomID: "general",
                UserID: user.ID,
                Content: message.Content,
                Timestamp: time.Now(),
            })
            
            // Broadcast to room
            chatRoom.Broadcast(web.Message{
                Type: "newMessage",
                Data: savedMessage,
            })
        }
        
        // Handle disconnect
        chatRoom.Leave(client.ID)
        chatRoom.Broadcast(web.Message{
            Type: "userLeft",
            Data: map[string]interface{}{
                "userID": user.ID,
                "username": user.Username,
            },
        })
    })
    
    // Real-time notifications
    http.HandleFunc("/ws/notifications", func(w http.ResponseWriter, r *http.Request) {
        conn, err := wsManager.Upgrade(w, r)
        if err != nil {
            return
        }
        
        userID := getUserIDFromToken(r.URL.Query().Get("token"))
        
        // Subscribe to user-specific notifications
        notifications := notificationService.Subscribe(userID)
        
        for notification := range notifications {
            conn.WriteJSON(web.Message{
                Type: "notification",
                Data: notification,
            })
        }
    })
}
```

---

*This comprehensive guide covers the major features of the Orizon Standard Library. Each section provides practical examples and best practices for building enterprise-grade applications.*
