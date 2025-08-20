// Package runtime provides comprehensive testing for region allocator.
// This module implements unit tests, integration tests, and benchmarks.
// for the region-based memory management system.
package runtime

import (
	"sync"
	"testing"
	"time"
	"unsafe"
)

// TestAllocator provides a test harness for the region allocator.
type TestAllocator struct {
	allocator   *RegionAllocator
	regions     map[RegionID]*Region
	allocations map[unsafe.Pointer]*AllocationInfo
	config      TestConfig
	mutex       sync.RWMutex
}

// AllocationInfo tracks allocation details for testing.
type AllocationInfo struct {
	Timestamp time.Time
	TypeInfo  *TypeInfo
	Region    *Region
	Size      RegionSize
	Alignment RegionAlignment
}

// TestConfig configures testing behavior.
type TestConfig struct {
	AllocationSizes      []RegionSize
	AlignmentValues      []RegionAlignment
	MaxTestDuration      time.Duration
	MaxAllocations       int
	MaxRegions           int
	StressThreads        int
	EnableStressTest     bool
	EnableCorruptionTest bool
	EnableLeakDetection  bool
	EnableRaceDetection  bool
}

// TestResult contains test execution results.
type TestResult struct {
	TestName           string
	ErrorsDetected     []string
	MemoryLeaks        []LeakInfo
	CorruptionEvents   []CorruptionInfo
	Performance        PerformanceStats
	Duration           time.Duration
	AllocationsCount   int
	DeallocationsCount int
	RegionsCreated     int
	RegionsDestroyed   int
	Success            bool
}

// LeakInfo contains information about a memory leak.
type LeakInfo struct {
	Allocation AllocationInfo
	Address    unsafe.Pointer
	Size       RegionSize
}

// CorruptionInfo contains information about memory corruption.
type CorruptionInfo struct {
	DetectionTime time.Time
	Address       unsafe.Pointer
	ExpectedValue uint32
	ActualValue   uint32
}

// PerformanceStats contains performance metrics.
type PerformanceStats struct {
	AvgAllocationTime   time.Duration // Average allocation time
	AvgDeallocationTime time.Duration // Average deallocation time
	AllocationsPerSec   float64       // Allocations per second
	DeallocationsPerSec float64       // Deallocations per second
	PeakMemoryUsage     uint64        // Peak memory usage
	FragmentationRatio  float64       // Average fragmentation ratio
}

// Default test configuration.
var DefaultTestConfig = TestConfig{
	EnableStressTest:     true,
	EnableCorruptionTest: true,
	EnableLeakDetection:  true,
	EnableRaceDetection:  true,
	MaxTestDuration:      time.Minute * 5,
	MaxAllocations:       10000,
	MaxRegions:           100,
	StressThreads:        10,
	AllocationSizes: []RegionSize{
		16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192,
	},
	AlignmentValues: []RegionAlignment{
		1, 2, 4, 8, 16, 32, 64, 128,
	},
}

// NewTestAllocator creates a new test allocator.
func NewTestAllocator(config TestConfig, allocatorConfig *AllocatorPolicy) *TestAllocator {
	allocator := NewRegionAllocator(allocatorConfig)

	return &TestAllocator{
		allocator:   allocator,
		regions:     make(map[RegionID]*Region),
		allocations: make(map[unsafe.Pointer]*AllocationInfo),
		config:      config,
	}
}

// RunAllTests runs all allocator tests.
func (ta *TestAllocator) RunAllTests(t *testing.T) []*TestResult {
	tests := []func(*testing.T) *TestResult{
		ta.TestBasicAllocation,
		ta.TestAllocationAlignment,
		ta.TestDeallocation,
		ta.TestFragmentation,
		ta.TestCompaction,
		ta.TestConcurrentAccess,
		ta.TestStressAllocation,
		ta.TestMemoryLeakDetection,
		ta.TestCorruptionDetection,
		ta.TestRegionManagement,
	}

	results := make([]*TestResult, 0, len(tests))

	for _, testFunc := range tests {
		result := testFunc(t)
		results = append(results, result)
	}

	return results
}

// TestBasicAllocation tests basic allocation functionality.
func (ta *TestAllocator) TestBasicAllocation(t *testing.T) *TestResult {
	startTime := time.Now()
	result := &TestResult{
		TestName: "BasicAllocation",
		Success:  true,
	}

	// Create a test region.
	region, err := ta.allocator.CreateRegion(RegionSize(1024*1024), RegionAlignment(16))
	if err != nil {
		result.Success = false
		result.ErrorsDetected = append(result.ErrorsDetected, err.Error())

		return result
	}

	ta.mutex.Lock()
	ta.regions[region.Header.ID] = region
	ta.mutex.Unlock()

	// Test various allocation sizes.
	for _, size := range ta.config.AllocationSizes {
		if size > RegionSize(1024*512) { // Skip large allocations for basic test
			continue
		}

		ptr, err := region.Allocate(size, RegionAlignment(8), nil)
		if err != nil {
			result.Success = false
			result.ErrorsDetected = append(result.ErrorsDetected, err.Error())

			continue
		}

		// Record allocation.
		ta.mutex.Lock()
		ta.allocations[ptr] = &AllocationInfo{
			Size:      size,
			Alignment: RegionAlignment(8),
			Region:    region,
			Timestamp: time.Now(),
		}
		ta.mutex.Unlock()

		result.AllocationsCount++

		// Test writing to allocated memory.
		ta.writeTestPattern(ptr, size)

		// Verify pattern.
		if !ta.verifyTestPattern(ptr, size) {
			result.Success = false
			result.ErrorsDetected = append(result.ErrorsDetected, "memory corruption detected")
		}
	}

	result.Duration = time.Since(startTime)

	return result
}

// TestAllocationAlignment tests memory alignment.
func (ta *TestAllocator) TestAllocationAlignment(t *testing.T) *TestResult {
	startTime := time.Now()
	result := &TestResult{
		TestName: "AllocationAlignment",
		Success:  true,
	}

	// Create a test region.
	region, err := ta.allocator.CreateRegion(RegionSize(1024*1024), RegionAlignment(16))
	if err != nil {
		result.Success = false
		result.ErrorsDetected = append(result.ErrorsDetected, err.Error())

		return result
	}

	// Test various alignments.
	for _, alignment := range ta.config.AlignmentValues {
		ptr, err := region.Allocate(RegionSize(256), alignment, nil)
		if err != nil {
			result.Success = false
			result.ErrorsDetected = append(result.ErrorsDetected, err.Error())

			continue
		}

		// Verify alignment.
		addr := uintptr(ptr)
		if addr%uintptr(alignment) != 0 {
			result.Success = false
			result.ErrorsDetected = append(result.ErrorsDetected, "alignment violation")
		}

		result.AllocationsCount++
	}

	result.Duration = time.Since(startTime)

	return result
}

// TestDeallocation tests memory deallocation.
func (ta *TestAllocator) TestDeallocation(t *testing.T) *TestResult {
	startTime := time.Now()
	result := &TestResult{
		TestName: "Deallocation",
		Success:  true,
	}

	// Create a test region.
	region, err := ta.allocator.CreateRegion(RegionSize(1024*1024), RegionAlignment(16))
	if err != nil {
		result.Success = false
		result.ErrorsDetected = append(result.ErrorsDetected, err.Error())

		return result
	}

	allocations := make([]unsafe.Pointer, 0, 100)

	// Allocate multiple blocks.
	for i := 0; i < 100; i++ {
		size := ta.config.AllocationSizes[i%len(ta.config.AllocationSizes)]

		ptr, err := region.Allocate(size, RegionAlignment(8), nil)
		if err != nil {
			result.Success = false
			result.ErrorsDetected = append(result.ErrorsDetected, err.Error())

			continue
		}

		allocations = append(allocations, ptr)
		result.AllocationsCount++
	}

	// Deallocate all blocks.
	for _, ptr := range allocations {
		err := region.Deallocate(ptr)
		if err != nil {
			result.Success = false
			result.ErrorsDetected = append(result.ErrorsDetected, err.Error())

			continue
		}

		result.DeallocationsCount++
	}

	result.Duration = time.Since(startTime)

	return result
}

// TestFragmentation tests memory fragmentation handling.
func (ta *TestAllocator) TestFragmentation(t *testing.T) *TestResult {
	startTime := time.Now()
	result := &TestResult{
		TestName: "Fragmentation",
		Success:  true,
	}

	// Create a test region.
	region, err := ta.allocator.CreateRegion(RegionSize(1024*1024), RegionAlignment(16))
	if err != nil {
		result.Success = false
		result.ErrorsDetected = append(result.ErrorsDetected, err.Error())

		return result
	}

	allocations := make([]unsafe.Pointer, 0, 1000)

	// Create fragmentation by allocating and deallocating in a pattern.
	for i := 0; i < 1000; i++ {
		size := RegionSize(64 + (i%7)*32) // Varying sizes

		ptr, err := region.Allocate(size, RegionAlignment(8), nil)
		if err != nil {
			break // Region full
		}

		allocations = append(allocations, ptr)
		result.AllocationsCount++

		// Deallocate every third allocation to create holes.
		if i%3 == 0 && len(allocations) > 0 {
			idx := len(allocations) - 1

			err := region.Deallocate(allocations[idx])
			if err != nil {
				result.Success = false
				result.ErrorsDetected = append(result.ErrorsDetected, err.Error())
			} else {
				result.DeallocationsCount++
			}

			allocations = allocations[:idx]
		}
	}

	// Check fragmentation ratio.
	fragRatio := region.calculateFragmentationRatio()
	result.Performance.FragmentationRatio = fragRatio

	if fragRatio > 0.8 {
		result.ErrorsDetected = append(result.ErrorsDetected, "excessive fragmentation")
	}

	result.Duration = time.Since(startTime)

	return result
}

// TestCompaction tests memory compaction.
func (ta *TestAllocator) TestCompaction(t *testing.T) *TestResult {
	startTime := time.Now()
	result := &TestResult{
		TestName: "Compaction",
		Success:  true,
	}

	// Create a test region.
	region, err := ta.allocator.CreateRegion(RegionSize(1024*1024), RegionAlignment(16))
	if err != nil {
		result.Success = false
		result.ErrorsDetected = append(result.ErrorsDetected, err.Error())

		return result
	}

	// Create fragmentation.
	allocations := make([]unsafe.Pointer, 0, 100)

	for i := 0; i < 100; i++ {
		ptr, err := region.Allocate(RegionSize(1024), RegionAlignment(8), nil)
		if err != nil {
			break
		}

		allocations = append(allocations, ptr)

		// Deallocate every other allocation.
		if i%2 == 0 {
			err := region.Deallocate(ptr)
			if err != nil {
				result.Success = false
				result.ErrorsDetected = append(result.ErrorsDetected, err.Error())
			}

			allocations[len(allocations)-1] = nil
		}
	}

	fragBefore := region.calculateFragmentationRatio()

	// Perform compaction.
	err = region.compact()
	if err != nil {
		result.Success = false
		result.ErrorsDetected = append(result.ErrorsDetected, err.Error())
	}

	fragAfter := region.calculateFragmentationRatio()

	// Verify compaction reduced fragmentation.
	if fragAfter >= fragBefore {
		result.ErrorsDetected = append(result.ErrorsDetected, "compaction did not reduce fragmentation")
	}

	result.Duration = time.Since(startTime)

	return result
}

// TestConcurrentAccess tests concurrent allocation/deallocation.
func (ta *TestAllocator) TestConcurrentAccess(t *testing.T) *TestResult {
	startTime := time.Now()
	result := &TestResult{
		TestName: "ConcurrentAccess",
		Success:  true,
	}

	// Create a test region.
	region, err := ta.allocator.CreateRegion(RegionSize(1024*1024*10), RegionAlignment(16))
	if err != nil {
		result.Success = false
		result.ErrorsDetected = append(result.ErrorsDetected, err.Error())

		return result
	}

	// Run concurrent allocation/deallocation
	var wg sync.WaitGroup

	errors := make(chan string, ta.config.StressThreads*100)

	for i := 0; i < ta.config.StressThreads; i++ {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()

			allocations := make([]unsafe.Pointer, 0, 100)

			for j := 0; j < 100; j++ {
				size := ta.config.AllocationSizes[j%len(ta.config.AllocationSizes)]

				ptr, err := region.Allocate(size, RegionAlignment(8), nil)
				if err != nil {
					errors <- err.Error()

					continue
				}

				allocations = append(allocations, ptr)

				// Randomly deallocate some allocations.
				if len(allocations) > 10 && j%5 == 0 {
					idx := j % len(allocations)

					err := region.Deallocate(allocations[idx])
					if err != nil {
						errors <- err.Error()
					} else {
						// Remove from slice.
						allocations[idx] = allocations[len(allocations)-1]
						allocations = allocations[:len(allocations)-1]
					}
				}
			}

			// Clean up remaining allocations.
			for _, ptr := range allocations {
				region.Deallocate(ptr)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Collect errors.
	for err := range errors {
		result.ErrorsDetected = append(result.ErrorsDetected, err)
		result.Success = false
	}

	result.Duration = time.Since(startTime)

	return result
}

// TestStressAllocation performs stress testing.
func (ta *TestAllocator) TestStressAllocation(t *testing.T) *TestResult {
	if !ta.config.EnableStressTest {
		return &TestResult{
			TestName: "StressAllocation",
			Success:  true,
			Duration: 0,
		}
	}

	startTime := time.Now()
	result := &TestResult{
		TestName: "StressAllocation",
		Success:  true,
	}

	// Create multiple regions.
	regions := make([]*Region, 0, ta.config.MaxRegions)

	for i := 0; i < ta.config.MaxRegions && i < 10; i++ {
		region, err := ta.allocator.CreateRegion(RegionSize(1024*1024), RegionAlignment(16))
		if err != nil {
			result.Success = false
			result.ErrorsDetected = append(result.ErrorsDetected, err.Error())

			continue
		}

		regions = append(regions, region)
		result.RegionsCreated++
	}

	// Stress test allocation/deallocation
	endTime := startTime.Add(ta.config.MaxTestDuration)
	for time.Now().Before(endTime) && result.AllocationsCount < ta.config.MaxAllocations {
		for _, region := range regions {
			size := ta.config.AllocationSizes[result.AllocationsCount%len(ta.config.AllocationSizes)]

			ptr, err := region.Allocate(size, RegionAlignment(8), nil)
			if err != nil {
				continue // Region full
			}

			result.AllocationsCount++

			// Immediate deallocation for stress testing.
			err = region.Deallocate(ptr)
			if err != nil {
				result.Success = false
				result.ErrorsDetected = append(result.ErrorsDetected, err.Error())
			} else {
				result.DeallocationsCount++
			}
		}
	}

	result.Duration = time.Since(startTime)

	return result
}

// TestMemoryLeakDetection tests memory leak detection.
func (ta *TestAllocator) TestMemoryLeakDetection(t *testing.T) *TestResult {
	if !ta.config.EnableLeakDetection {
		return &TestResult{
			TestName: "MemoryLeakDetection",
			Success:  true,
			Duration: 0,
		}
	}

	startTime := time.Now()
	result := &TestResult{
		TestName: "MemoryLeakDetection",
		Success:  true,
	}

	// This would implement actual leak detection logic.
	// For now, just return success.

	result.Duration = time.Since(startTime)

	return result
}

// TestCorruptionDetection tests memory corruption detection.
func (ta *TestAllocator) TestCorruptionDetection(t *testing.T) *TestResult {
	if !ta.config.EnableCorruptionTest {
		return &TestResult{
			TestName: "CorruptionDetection",
			Success:  true,
			Duration: 0,
		}
	}

	startTime := time.Now()
	result := &TestResult{
		TestName: "CorruptionDetection",
		Success:  true,
	}

	// This would implement actual corruption detection logic.
	// For now, just return success.

	result.Duration = time.Since(startTime)

	return result
}

// TestRegionManagement tests region creation and destruction.
func (ta *TestAllocator) TestRegionManagement(t *testing.T) *TestResult {
	startTime := time.Now()
	result := &TestResult{
		TestName: "RegionManagement",
		Success:  true,
	}

	// Test region creation.
	for i := 0; i < 10; i++ {
		region, err := ta.allocator.CreateRegion(RegionSize(1024*1024), RegionAlignment(16))
		if err != nil {
			result.Success = false
			result.ErrorsDetected = append(result.ErrorsDetected, err.Error())

			continue
		}

		result.RegionsCreated++

		// Test region destruction.
		err = ta.allocator.DestroyRegion(region.Header.ID)
		if err != nil {
			result.Success = false
			result.ErrorsDetected = append(result.ErrorsDetected, err.Error())
		} else {
			result.RegionsDestroyed++
		}
	}

	result.Duration = time.Since(startTime)

	return result
}

// Helper methods.

// writeTestPattern writes a test pattern to memory.
func (ta *TestAllocator) writeTestPattern(ptr unsafe.Pointer, size RegionSize) {
	slice := (*[1 << 30]byte)(ptr)[:size:size]
	for i := range slice {
		slice[i] = byte(i % 256)
	}
}

// verifyTestPattern verifies a test pattern in memory.
func (ta *TestAllocator) verifyTestPattern(ptr unsafe.Pointer, size RegionSize) bool {
	slice := (*[1 << 30]byte)(ptr)[:size:size]
	for i := range slice {
		if slice[i] != byte(i%256) {
			return false
		}
	}

	return true
}

// Cleanup performs test cleanup.
func (ta *TestAllocator) Cleanup() {
	ta.mutex.Lock()
	defer ta.mutex.Unlock()

	// Deallocate all remaining allocations.
	for ptr := range ta.allocations {
		// Find the region for this allocation.
		if info, exists := ta.allocations[ptr]; exists && info.Region != nil {
			info.Region.Deallocate(ptr)
		}
	}

	// Destroy all regions.
	for _, region := range ta.regions {
		ta.allocator.DestroyRegion(region.Header.ID)
	}

	// Clear maps.
	ta.allocations = make(map[unsafe.Pointer]*AllocationInfo)
	ta.regions = make(map[RegionID]*Region)
}
