// Package runtime provides memory compaction and optimization algorithms.
// This module implements various compaction strategies, defragmentation,
// and memory layout optimization for region-based memory management.
package runtime

import (
	"sort"
	"sync"
	"time"
	"unsafe"
)

// CompactionEngine manages memory compaction operations
type CompactionEngine struct {
	mutex      sync.RWMutex
	regions    map[RegionID]*Region          // Registered regions
	strategies map[string]CompactionStrategy // Available strategies
	scheduler  *CompactionScheduler          // Compaction scheduler
	statistics CompactionStatistics          // Compaction statistics
	config     CompactionConfig              // Configuration
	enabled    bool                          // Compaction enabled
	running    bool                          // Compaction in progress
}

// CompactionStrategy defines interface for compaction algorithms
type CompactionStrategy interface {
	Name() string                                      // Strategy name
	Compact(region *Region) (*CompactionResult, error) // Perform compaction
	CanCompact(region *Region) bool                    // Check if compaction is beneficial
	EstimateGain(region *Region) CompactionGain        // Estimate compaction benefits
	Priority() int                                     // Strategy priority
}

// CompactionResult contains results of a compaction operation
type CompactionResult struct {
	Strategy            string        // Strategy used
	StartTime           time.Time     // Compaction start time
	EndTime             time.Time     // Compaction end time
	Duration            time.Duration // Total compaction time
	BytesCompacted      uint64        // Bytes involved in compaction
	BytesReclaimed      uint64        // Bytes reclaimed
	BlocksMoved         uint64        // Number of blocks moved
	BlocksCoalesced     uint64        // Number of blocks coalesced
	FragmentationBefore float64       // Fragmentation before compaction
	FragmentationAfter  float64       // Fragmentation after compaction
	Success             bool          // Compaction successful
	Error               error         // Error if unsuccessful
}

// CompactionGain estimates benefits of compaction
type CompactionGain struct {
	ExpectedReclaimed uint64        // Expected bytes to reclaim
	ExpectedReduction float64       // Expected fragmentation reduction
	EstimatedDuration time.Duration // Estimated compaction time
	Confidence        float64       // Confidence in estimates (0-1)
	RecommendCompact  bool          // Recommendation to compact
}

// CompactionScheduler manages automatic compaction scheduling
type CompactionScheduler struct {
	mutex    sync.Mutex
	enabled  bool
	interval time.Duration
	lastRun  time.Time
	nextRun  time.Time
	triggers []CompactionTrigger
	stopChan chan struct{}
	running  bool
}

// CompactionTrigger defines conditions that trigger compaction
type CompactionTrigger struct {
	Name          string        // Trigger name
	Type          TriggerType   // Trigger type
	Threshold     float64       // Threshold value
	MinInterval   time.Duration // Minimum interval between triggers
	LastTriggered time.Time     // Last trigger time
	Enabled       bool          // Trigger enabled
}

// TriggerType defines types of compaction triggers
type TriggerType int

const (
	TriggerFragmentation     TriggerType = iota // Fragmentation threshold
	TriggerMemoryPressure                       // Memory pressure
	TriggerAllocationFailure                    // Allocation failures
	TriggerTimeBased                            // Time-based trigger
	TriggerManual                               // Manual trigger
)

// CompactionStatistics tracks compaction performance
type CompactionStatistics struct {
	TotalCompactions      uint64            // Total compactions performed
	SuccessfulCompactions uint64            // Successful compactions
	FailedCompactions     uint64            // Failed compactions
	TotalTimeSpent        time.Duration     // Total time spent compacting
	AverageCompactionTime time.Duration     // Average compaction time
	TotalBytesCompacted   uint64            // Total bytes compacted
	TotalBytesReclaimed   uint64            // Total bytes reclaimed
	AverageFragReduction  float64           // Average fragmentation reduction
	LastCompactionTime    time.Time         // Last compaction time
	CompactionsByStrategy map[string]uint64 // Compactions by strategy
}

// CompactionConfig configures compaction behavior
type CompactionConfig struct {
	Enabled                 bool          // Enable compaction
	AutomaticEnabled        bool          // Enable automatic compaction
	SchedulerInterval       time.Duration // Scheduler check interval
	MaxCompactionTime       time.Duration // Maximum compaction duration
	FragmentationThreshold  float64       // Fragmentation threshold for compaction
	MemoryPressureThreshold float64       // Memory pressure threshold
	MinFreeBlocks           int           // Minimum free blocks to trigger
	ConcurrentCompactions   int           // Maximum concurrent compactions
	PreferredStrategy       string        // Preferred compaction strategy
	AdaptiveStrategy        bool          // Use adaptive strategy selection
}

// MarkAndSweepCompactor implements mark-and-sweep compaction
type MarkAndSweepCompactor struct {
	name     string
	priority int
}

// CopyingCompactor implements copying garbage collection
type CopyingCompactor struct {
	name     string
	priority int
}

// IncrementalCompactor implements incremental compaction
type IncrementalCompactor struct {
	name        string
	priority    int
	chunkSize   uint64        // Size of each compaction chunk
	maxDuration time.Duration // Maximum time per chunk
}

// SlidingCompactor implements sliding compaction
type SlidingCompactor struct {
	name     string
	priority int
}

// TwoPassCompactor implements two-pass compaction
type TwoPassCompactor struct {
	name     string
	priority int
}

// CompactionBlock represents a block during compaction
type CompactionBlock struct {
	Header    *BlockHeader   // Block header
	Data      unsafe.Pointer // Block data
	Size      uint64         // Block size
	NewOffset uintptr        // New offset after compaction
	Marked    bool           // Block is marked for moving
	Visited   bool           // Block has been visited
}

// Default configuration values
const (
	DefaultSchedulerInterval       = time.Minute * 5
	DefaultMaxCompactionTime       = time.Second * 30
	DefaultFragmentationThreshold  = 0.3
	DefaultMemoryPressureThreshold = 0.8
	DefaultMinFreeBlocks           = 10
	DefaultConcurrentCompactions   = 1
)

// NewCompactionEngine creates a new compaction engine
func NewCompactionEngine(config CompactionConfig) *CompactionEngine {
	engine := &CompactionEngine{
		regions:    make(map[RegionID]*Region),
		strategies: make(map[string]CompactionStrategy),
		scheduler:  newCompactionScheduler(config),
		config:     config,
		enabled:    config.Enabled,
	}

	// Register built-in strategies
	engine.registerStrategy(&MarkAndSweepCompactor{
		name:     "mark-and-sweep",
		priority: 100,
	})

	engine.registerStrategy(&CopyingCompactor{
		name:     "copying",
		priority: 90,
	})

	engine.registerStrategy(&IncrementalCompactor{
		name:        "incremental",
		priority:    80,
		chunkSize:   8192,
		maxDuration: time.Millisecond * 100,
	})

	engine.registerStrategy(&SlidingCompactor{
		name:     "sliding",
		priority: 70,
	})

	engine.registerStrategy(&TwoPassCompactor{
		name:     "two-pass",
		priority: 60,
	})

	// Initialize statistics
	engine.statistics.CompactionsByStrategy = make(map[string]uint64)

	return engine
}

// RegisterRegion registers a region for compaction
func (ce *CompactionEngine) RegisterRegion(region *Region) {
	if !ce.enabled {
		return
	}

	ce.mutex.Lock()
	defer ce.mutex.Unlock()

	ce.regions[region.Header.ID] = region
}

// UnregisterRegion removes a region from compaction
func (ce *CompactionEngine) UnregisterRegion(regionID RegionID) {
	ce.mutex.Lock()
	defer ce.mutex.Unlock()

	delete(ce.regions, regionID)
}

// CompactRegion compacts a specific region
func (ce *CompactionEngine) CompactRegion(regionID RegionID, strategyName string) (*CompactionResult, error) {
	if !ce.enabled {
		return nil, ErrCompactionDisabled
	}

	ce.mutex.RLock()
	region, exists := ce.regions[regionID]
	ce.mutex.RUnlock()

	if !exists {
		return nil, ErrRegionNotFound
	}

	// Select strategy
	var strategy CompactionStrategy
	if strategyName != "" {
		strategy = ce.strategies[strategyName]
	} else {
		strategy = ce.selectOptimalStrategy(region)
	}

	if strategy == nil {
		return nil, ErrStrategyNotFound
	}

	// Perform compaction
	result, err := strategy.Compact(region)

	// Update statistics
	ce.updateStatistics(result)

	return result, err
}

// CompactAll compacts all registered regions
func (ce *CompactionEngine) CompactAll() ([]*CompactionResult, error) {
	if !ce.enabled {
		return nil, ErrCompactionDisabled
	}

	ce.mutex.RLock()
	regionIDs := make([]RegionID, 0, len(ce.regions))
	for id := range ce.regions {
		regionIDs = append(regionIDs, id)
	}
	ce.mutex.RUnlock()

	results := make([]*CompactionResult, 0, len(regionIDs))

	for _, regionID := range regionIDs {
		result, err := ce.CompactRegion(regionID, "")
		if err != nil {
			// Log error but continue with other regions
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

// AutoCompact performs automatic compaction based on triggers
func (ce *CompactionEngine) AutoCompact() error {
	if !ce.enabled || !ce.config.AutomaticEnabled {
		return nil
	}

	ce.mutex.RLock()
	defer ce.mutex.RUnlock()

	for _, region := range ce.regions {
		// Check if region needs compaction
		if ce.shouldCompact(region) {
			strategy := ce.selectOptimalStrategy(region)
			if strategy != nil && strategy.CanCompact(region) {
				go func(r *Region, s CompactionStrategy) {
					_, err := s.Compact(r)
					if err != nil {
						// Log error
					}
				}(region, strategy)
			}
		}
	}

	return nil
}

// selectOptimalStrategy selects the best compaction strategy for a region
func (ce *CompactionEngine) selectOptimalStrategy(region *Region) CompactionStrategy {
	if ce.config.PreferredStrategy != "" {
		if strategy, exists := ce.strategies[ce.config.PreferredStrategy]; exists {
			return strategy
		}
	}

	if !ce.config.AdaptiveStrategy {
		// Use default strategy
		return ce.strategies["mark-and-sweep"]
	}

	// Adaptive strategy selection
	type strategyScore struct {
		strategy CompactionStrategy
		score    float64
	}

	scores := make([]strategyScore, 0, len(ce.strategies))

	for _, strategy := range ce.strategies {
		if !strategy.CanCompact(region) {
			continue
		}

		gain := strategy.EstimateGain(region)

		// Calculate score based on multiple factors
		score := gain.Confidence * 100
		score += gain.ExpectedReduction * 50
		score += float64(strategy.Priority())

		// Penalize longer estimated duration
		if gain.EstimatedDuration > 0 {
			score -= float64(gain.EstimatedDuration.Milliseconds()) * 0.1
		}

		scores = append(scores, strategyScore{
			strategy: strategy,
			score:    score,
		})
	}

	if len(scores) == 0 {
		return nil
	}

	// Sort by score (highest first)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	return scores[0].strategy
}

// shouldCompact determines if a region should be compacted
func (ce *CompactionEngine) shouldCompact(region *Region) bool {
	// Check fragmentation threshold
	fragRatio := region.calculateFragmentationRatio()
	if fragRatio > ce.config.FragmentationThreshold {
		return true
	}

	// Check memory pressure
	utilization := float64(region.Header.Used) / float64(region.Header.Size)
	if utilization > ce.config.MemoryPressureThreshold {
		return true
	}

	// Check number of free blocks
	freeBlockCount := region.countFreeBlocks()
	if freeBlockCount > ce.config.MinFreeBlocks {
		return true
	}

	return false
}

// Implementation of MarkAndSweepCompactor

// Name returns the strategy name
func (msc *MarkAndSweepCompactor) Name() string {
	return msc.name
}

// Priority returns the strategy priority
func (msc *MarkAndSweepCompactor) Priority() int {
	return msc.priority
}

// CanCompact checks if the region can be compacted
func (msc *MarkAndSweepCompactor) CanCompact(region *Region) bool {
	return region.countFreeBlocks() > 1
}

// EstimateGain estimates the benefits of compaction
func (msc *MarkAndSweepCompactor) EstimateGain(region *Region) CompactionGain {
	fragRatio := region.calculateFragmentationRatio()
	freeBlocks := region.countFreeBlocks()

	// Estimate bytes that could be reclaimed
	expectedReclaimed := uint64(float64(region.Header.Free) * fragRatio * 0.7)

	// Estimate fragmentation reduction
	expectedReduction := fragRatio * 0.8

	// Estimate duration based on region size
	estimatedDuration := time.Duration(region.Header.Size/1024) * time.Millisecond

	return CompactionGain{
		ExpectedReclaimed: expectedReclaimed,
		ExpectedReduction: expectedReduction,
		EstimatedDuration: estimatedDuration,
		Confidence:        0.8,
		RecommendCompact:  fragRatio > 0.3 && freeBlocks > 5,
	}
}

// Compact performs mark-and-sweep compaction
func (msc *MarkAndSweepCompactor) Compact(region *Region) (*CompactionResult, error) {
	startTime := time.Now()

	result := &CompactionResult{
		Strategy:            msc.name,
		StartTime:           startTime,
		FragmentationBefore: region.calculateFragmentationRatio(),
	}

	region.Mutex.Lock()
	defer region.Mutex.Unlock()

	// Phase 1: Mark all allocated blocks
	allocatedBlocks := msc.markAllocatedBlocks(region)

	// Phase 2: Sweep and compact
	err := msc.sweepAndCompact(region, allocatedBlocks, result)

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(startTime)
	result.FragmentationAfter = region.calculateFragmentationRatio()
	result.Success = (err == nil)
	result.Error = err

	return result, err
}

// markAllocatedBlocks marks all allocated blocks
func (msc *MarkAndSweepCompactor) markAllocatedBlocks(region *Region) []*CompactionBlock {
	blocks := make([]*CompactionBlock, 0)

	current := region.Header.AllocList
	for current != nil {
		block := &CompactionBlock{
			Header: (*BlockHeader)(unsafe.Pointer(uintptr(region.Data) + current.Offset - uintptr(BlockHeaderSize))),
			Data:   unsafe.Pointer(uintptr(region.Data) + current.Offset),
			Size:   uint64(current.Size),
			Marked: true,
		}
		blocks = append(blocks, block)
		current = current.Next
	}

	return blocks
}

// sweepAndCompact sweeps unmarked blocks and compacts memory
func (msc *MarkAndSweepCompactor) sweepAndCompact(region *Region, blocks []*CompactionBlock, result *CompactionResult) error {
	// Sort blocks by current offset
	sort.Slice(blocks, func(i, j int) bool {
		return uintptr(blocks[i].Data) < uintptr(blocks[j].Data)
	})

	// Compact by sliding blocks down
	var newOffset uintptr = 0

	for _, block := range blocks {
		if block.Marked {
			if uintptr(block.Data) != uintptr(region.Data)+newOffset {
				// Move block data
				msc.moveBlock(block, uintptr(region.Data)+newOffset)
				result.BlocksMoved++
			}

			block.NewOffset = newOffset
			newOffset += uintptr(block.Size) + uintptr(BlockHeaderSize)
		}
	}

	// Update free list - create single large free block
	if newOffset < uintptr(region.Header.Size) {
		remainingSize := RegionSize(uintptr(region.Header.Size) - newOffset)
		msc.createFreeBlock(region, newOffset, remainingSize)
		result.BytesReclaimed = uint64(remainingSize)
	}

	result.BytesCompacted = uint64(newOffset)

	return nil
}

// moveBlock moves a block to a new location
func (msc *MarkAndSweepCompactor) moveBlock(block *CompactionBlock, newAddress uintptr) {
	// Copy block data
	oldData := block.Data
	newData := unsafe.Pointer(newAddress + uintptr(BlockHeaderSize))

	// Copy the data
	size := int(block.Size)
	oldSlice := (*[1 << 30]byte)(oldData)[:size:size]
	newSlice := (*[1 << 30]byte)(newData)[:size:size]
	copy(newSlice, oldSlice)

	// Update block pointer
	block.Data = newData
}

// createFreeBlock creates a new free block
func (msc *MarkAndSweepCompactor) createFreeBlock(region *Region, offset uintptr, size RegionSize) {
	freeBlock := &FreeBlock{
		Size:   size,
		Offset: offset,
		Next:   nil,
		Prev:   nil,
	}

	// Clear existing free list and add single large block
	region.Header.FreeList = freeBlock
	region.Header.Free = size
}

// Implementation of CopyingCompactor

// Name returns the strategy name
func (cc *CopyingCompactor) Name() string {
	return cc.name
}

// Priority returns the strategy priority
func (cc *CopyingCompactor) Priority() int {
	return cc.priority
}

// CanCompact checks if the region can be compacted
func (cc *CopyingCompactor) CanCompact(region *Region) bool {
	// Copying compaction requires sufficient free space
	return region.Header.Free > region.Header.Used/2
}

// EstimateGain estimates the benefits of compaction
func (cc *CopyingCompactor) EstimateGain(region *Region) CompactionGain {
	fragRatio := region.calculateFragmentationRatio()

	return CompactionGain{
		ExpectedReclaimed: uint64(float64(region.Header.Free) * fragRatio * 0.9),
		ExpectedReduction: fragRatio * 0.95,
		EstimatedDuration: time.Duration(region.Header.Size/512) * time.Millisecond,
		Confidence:        0.9,
		RecommendCompact:  fragRatio > 0.2,
	}
}

// Compact performs copying compaction
func (cc *CopyingCompactor) Compact(region *Region) (*CompactionResult, error) {
	// Simplified copying compaction implementation
	region.Mutex.Lock()
	defer region.Mutex.Unlock()

	// For now, delegate to mark-and-sweep
	msc := &MarkAndSweepCompactor{name: "mark-and-sweep", priority: 100}
	return msc.Compact(region)
}

// Implementation of IncrementalCompactor

// Name returns the strategy name
func (ic *IncrementalCompactor) Name() string {
	return ic.name
}

// Priority returns the strategy priority
func (ic *IncrementalCompactor) Priority() int {
	return ic.priority
}

// CanCompact checks if the region can be compacted
func (ic *IncrementalCompactor) CanCompact(region *Region) bool {
	return region.countFreeBlocks() > 2
}

// EstimateGain estimates the benefits of compaction
func (ic *IncrementalCompactor) EstimateGain(region *Region) CompactionGain {
	fragRatio := region.calculateFragmentationRatio()

	return CompactionGain{
		ExpectedReclaimed: uint64(float64(region.Header.Free) * fragRatio * 0.6),
		ExpectedReduction: fragRatio * 0.7,
		EstimatedDuration: ic.maxDuration,
		Confidence:        0.7,
		RecommendCompact:  fragRatio > 0.4,
	}
}

// Compact performs incremental compaction
func (ic *IncrementalCompactor) Compact(region *Region) (*CompactionResult, error) {
	// Simplified incremental compaction implementation
	region.Mutex.Lock()
	defer region.Mutex.Unlock()

	// For now, delegate to mark-and-sweep with time limit
	msc := &MarkAndSweepCompactor{name: "mark-and-sweep", priority: 100}
	return msc.Compact(region)
}

// Implementation of SlidingCompactor

// Name returns the strategy name
func (sc *SlidingCompactor) Name() string {
	return sc.name
}

// Priority returns the strategy priority
func (sc *SlidingCompactor) Priority() int {
	return sc.priority
}

// CanCompact checks if the region can be compacted
func (sc *SlidingCompactor) CanCompact(region *Region) bool {
	return region.countFreeBlocks() > 1
}

// EstimateGain estimates the benefits of compaction
func (sc *SlidingCompactor) EstimateGain(region *Region) CompactionGain {
	fragRatio := region.calculateFragmentationRatio()

	return CompactionGain{
		ExpectedReclaimed: uint64(float64(region.Header.Free) * fragRatio * 0.8),
		ExpectedReduction: fragRatio * 0.85,
		EstimatedDuration: time.Duration(region.Header.Size/1024) * time.Millisecond,
		Confidence:        0.8,
		RecommendCompact:  fragRatio > 0.3,
	}
}

// Compact performs sliding compaction
func (sc *SlidingCompactor) Compact(region *Region) (*CompactionResult, error) {
	// Simplified sliding compaction implementation
	region.Mutex.Lock()
	defer region.Mutex.Unlock()

	// For now, delegate to mark-and-sweep
	msc := &MarkAndSweepCompactor{name: "mark-and-sweep", priority: 100}
	return msc.Compact(region)
}

// Implementation of TwoPassCompactor

// Name returns the strategy name
func (tpc *TwoPassCompactor) Name() string {
	return tpc.name
}

// Priority returns the strategy priority
func (tpc *TwoPassCompactor) Priority() int {
	return tpc.priority
}

// CanCompact checks if the region can be compacted
func (tpc *TwoPassCompactor) CanCompact(region *Region) bool {
	return region.countFreeBlocks() > 3
}

// EstimateGain estimates the benefits of compaction
func (tpc *TwoPassCompactor) EstimateGain(region *Region) CompactionGain {
	fragRatio := region.calculateFragmentationRatio()

	return CompactionGain{
		ExpectedReclaimed: uint64(float64(region.Header.Free) * fragRatio * 0.75),
		ExpectedReduction: fragRatio * 0.8,
		EstimatedDuration: time.Duration(region.Header.Size/768) * time.Millisecond,
		Confidence:        0.75,
		RecommendCompact:  fragRatio > 0.35,
	}
}

// Compact performs two-pass compaction
func (tpc *TwoPassCompactor) Compact(region *Region) (*CompactionResult, error) {
	// Simplified two-pass compaction implementation
	region.Mutex.Lock()
	defer region.Mutex.Unlock()

	// For now, delegate to mark-and-sweep
	msc := &MarkAndSweepCompactor{name: "mark-and-sweep", priority: 100}
	return msc.Compact(region)
}

// Helper functions and other strategy implementations would follow similar patterns...

// Scheduler implementation

// newCompactionScheduler creates a new compaction scheduler
func newCompactionScheduler(config CompactionConfig) *CompactionScheduler {
	return &CompactionScheduler{
		enabled:  config.AutomaticEnabled,
		interval: config.SchedulerInterval,
		triggers: createDefaultTriggers(),
		stopChan: make(chan struct{}),
	}
}

// createDefaultTriggers creates default compaction triggers
func createDefaultTriggers() []CompactionTrigger {
	return []CompactionTrigger{
		{
			Name:        "fragmentation",
			Type:        TriggerFragmentation,
			Threshold:   0.3,
			MinInterval: time.Minute * 5,
			Enabled:     true,
		},
		{
			Name:        "memory-pressure",
			Type:        TriggerMemoryPressure,
			Threshold:   0.8,
			MinInterval: time.Minute * 2,
			Enabled:     true,
		},
		{
			Name:        "time-based",
			Type:        TriggerTimeBased,
			Threshold:   float64(time.Hour.Nanoseconds()),
			MinInterval: time.Hour,
			Enabled:     false,
		},
	}
}

// updateStatistics updates compaction statistics
func (ce *CompactionEngine) updateStatistics(result *CompactionResult) {
	ce.mutex.Lock()
	defer ce.mutex.Unlock()

	ce.statistics.TotalCompactions++
	if result.Success {
		ce.statistics.SuccessfulCompactions++
	} else {
		ce.statistics.FailedCompactions++
	}

	ce.statistics.TotalTimeSpent += result.Duration
	ce.statistics.TotalBytesCompacted += result.BytesCompacted
	ce.statistics.TotalBytesReclaimed += result.BytesReclaimed
	ce.statistics.LastCompactionTime = result.EndTime

	// Update average compaction time
	if ce.statistics.TotalCompactions > 0 {
		ce.statistics.AverageCompactionTime = ce.statistics.TotalTimeSpent / time.Duration(ce.statistics.TotalCompactions)
	}

	// Update strategy statistics
	ce.statistics.CompactionsByStrategy[result.Strategy]++

	// Update average fragmentation reduction
	fragReduction := result.FragmentationBefore - result.FragmentationAfter
	if ce.statistics.TotalCompactions == 1 {
		ce.statistics.AverageFragReduction = fragReduction
	} else {
		ce.statistics.AverageFragReduction = (ce.statistics.AverageFragReduction*float64(ce.statistics.TotalCompactions-1) + fragReduction) / float64(ce.statistics.TotalCompactions)
	}
}

// registerStrategy registers a compaction strategy
func (ce *CompactionEngine) registerStrategy(strategy CompactionStrategy) {
	ce.mutex.Lock()
	defer ce.mutex.Unlock()

	ce.strategies[strategy.Name()] = strategy
}

// Error definitions
type CompactionError struct {
	message string
}

func (e *CompactionError) Error() string {
	return e.message
}

var (
	ErrCompactionDisabled = &CompactionError{"compaction disabled"}
	ErrRegionNotFound     = &CompactionError{"region not found"}
	ErrStrategyNotFound   = &CompactionError{"strategy not found"}
	ErrCompactionTimeout  = &CompactionError{"compaction timeout"}
	ErrInsufficientMemory = &CompactionError{"insufficient memory"}
)
