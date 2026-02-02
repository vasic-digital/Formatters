package cache

import (
	"testing"
	"time"

	"digital.vasic.formatters/pkg/formatter"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCache(ttl time.Duration, maxEntries int) *InMemoryCache {
	return NewInMemoryCache(Config{
		MaxEntries:  maxEntries,
		TTL:         ttl,
		CleanupFreq: 1 * time.Hour,
	})
}

func TestNewInMemoryCache(t *testing.T) {
	c := newTestCache(1*time.Hour, 100)
	defer c.Stop()

	assert.NotNil(t, c)
	assert.Equal(t, 0, c.Size())
}

func TestInMemoryCache_Set_Get(t *testing.T) {
	c := newTestCache(1*time.Hour, 100)
	defer c.Stop()

	req := &formatter.FormatRequest{
		Content:  "x = 1",
		Language: "python",
		FilePath: "test.py",
	}
	result := &formatter.FormatResult{
		Content:       "x = 1\n",
		Changed:       true,
		FormatterName: "black",
		Success:       true,
	}

	c.Set(req, result)

	got, found := c.Get(req)
	assert.True(t, found)
	require.NotNil(t, got)
	assert.Equal(t, "x = 1\n", got.Content)
	assert.Equal(t, "black", got.FormatterName)
	assert.True(t, got.Success)
}

func TestInMemoryCache_Get_Miss(t *testing.T) {
	c := newTestCache(1*time.Hour, 100)
	defer c.Stop()

	req := &formatter.FormatRequest{
		Content:  "not cached",
		Language: "python",
	}

	got, found := c.Get(req)
	assert.False(t, found)
	assert.Nil(t, got)
}

func TestInMemoryCache_Get_Expired(t *testing.T) {
	c := newTestCache(1*time.Millisecond, 100)
	defer c.Stop()

	req := &formatter.FormatRequest{
		Content:  "x = 1",
		Language: "python",
	}
	result := &formatter.FormatResult{
		Content: "x = 1\n",
		Success: true,
	}

	c.Set(req, result)

	time.Sleep(10 * time.Millisecond)

	got, found := c.Get(req)
	assert.False(t, found)
	assert.Nil(t, got)
}

func TestInMemoryCache_Set_Eviction(t *testing.T) {
	c := newTestCache(1*time.Hour, 2)
	defer c.Stop()

	for i := 0; i < 3; i++ {
		req := &formatter.FormatRequest{
			Content:  string(rune('a' + i)),
			Language: "python",
		}
		result := &formatter.FormatResult{
			Content: string(rune('a'+i)) + " formatted",
			Success: true,
		}
		c.Set(req, result)
		time.Sleep(1 * time.Millisecond)
	}

	assert.LessOrEqual(t, c.Size(), 2)
}

func TestInMemoryCache_Clear(t *testing.T) {
	c := newTestCache(1*time.Hour, 100)
	defer c.Stop()

	for i := 0; i < 5; i++ {
		req := &formatter.FormatRequest{
			Content:  string(rune('a' + i)),
			Language: "python",
		}
		result := &formatter.FormatResult{Content: "result"}
		c.Set(req, result)
	}

	assert.Equal(t, 5, c.Size())

	c.Clear()
	assert.Equal(t, 0, c.Size())
}

func TestInMemoryCache_Size(t *testing.T) {
	c := newTestCache(1*time.Hour, 100)
	defer c.Stop()

	assert.Equal(t, 0, c.Size())

	c.Set(
		&formatter.FormatRequest{Content: "a", Language: "go"},
		&formatter.FormatResult{Content: "a"},
	)
	assert.Equal(t, 1, c.Size())

	c.Set(
		&formatter.FormatRequest{Content: "b", Language: "go"},
		&formatter.FormatResult{Content: "b"},
	)
	assert.Equal(t, 2, c.Size())
}

func TestInMemoryCache_Stats(t *testing.T) {
	c := newTestCache(30*time.Minute, 500)
	defer c.Stop()

	c.Set(
		&formatter.FormatRequest{Content: "a"},
		&formatter.FormatResult{Content: "a"},
	)

	stats := c.Stats()
	assert.Equal(t, 1, stats.Size)
	assert.Equal(t, 500, stats.MaxEntries)
	assert.Equal(t, 30*time.Minute, stats.TTL)
}

func TestInMemoryCache_Stop(t *testing.T) {
	c := newTestCache(1*time.Hour, 100)
	c.Stop()
}

func TestInMemoryCache_Invalidate(t *testing.T) {
	c := newTestCache(1*time.Hour, 100)
	defer c.Stop()

	req := &formatter.FormatRequest{
		Content:  "x = 1",
		Language: "python",
	}
	result := &formatter.FormatResult{
		Content: "x = 1\n",
		Success: true,
	}

	c.Set(req, result)
	assert.Equal(t, 1, c.Size())

	c.Invalidate(req)
	assert.Equal(t, 0, c.Size())

	got, found := c.Get(req)
	assert.False(t, found)
	assert.Nil(t, got)
}

func TestInMemoryCache_CacheKey_DifferentContent(t *testing.T) {
	c := newTestCache(1*time.Hour, 100)
	defer c.Stop()

	req1 := &formatter.FormatRequest{
		Content: "x = 1", Language: "python",
	}
	req2 := &formatter.FormatRequest{
		Content: "y = 2", Language: "python",
	}

	result1 := &formatter.FormatResult{
		Content: "x = 1\n", Success: true,
	}
	result2 := &formatter.FormatResult{
		Content: "y = 2\n", Success: true,
	}

	c.Set(req1, result1)
	c.Set(req2, result2)

	got1, found1 := c.Get(req1)
	assert.True(t, found1)
	assert.Equal(t, "x = 1\n", got1.Content)

	got2, found2 := c.Get(req2)
	assert.True(t, found2)
	assert.Equal(t, "y = 2\n", got2.Content)
}

func TestInMemoryCache_CacheKey_DifferentLanguage(t *testing.T) {
	c := newTestCache(1*time.Hour, 100)
	defer c.Stop()

	req1 := &formatter.FormatRequest{
		Content: "code", Language: "python",
	}
	req2 := &formatter.FormatRequest{
		Content: "code", Language: "javascript",
	}

	result1 := &formatter.FormatResult{Content: "python formatted"}
	result2 := &formatter.FormatResult{Content: "js formatted"}

	c.Set(req1, result1)
	c.Set(req2, result2)

	got1, _ := c.Get(req1)
	assert.Equal(t, "python formatted", got1.Content)

	got2, _ := c.Get(req2)
	assert.Equal(t, "js formatted", got2.Content)
}

func TestInMemoryCache_CacheKey_DifferentFilePath(t *testing.T) {
	c := newTestCache(1*time.Hour, 100)
	defer c.Stop()

	req1 := &formatter.FormatRequest{
		Content: "code", FilePath: "a.py",
	}
	req2 := &formatter.FormatRequest{
		Content: "code", FilePath: "b.py",
	}

	result1 := &formatter.FormatResult{Content: "result a"}
	result2 := &formatter.FormatResult{Content: "result b"}

	c.Set(req1, result1)
	c.Set(req2, result2)

	got1, _ := c.Get(req1)
	assert.Equal(t, "result a", got1.Content)

	got2, _ := c.Get(req2)
	assert.Equal(t, "result b", got2.Content)
}

func TestInMemoryCache_Set_SameKeyOverwrites(t *testing.T) {
	c := newTestCache(1*time.Hour, 100)
	defer c.Stop()

	req := &formatter.FormatRequest{
		Content: "x = 1", Language: "python",
	}

	c.Set(req, &formatter.FormatResult{Content: "version1"})
	c.Set(req, &formatter.FormatResult{Content: "version2"})

	got, found := c.Get(req)
	assert.True(t, found)
	assert.Equal(t, "version2", got.Content)
	assert.Equal(t, 1, c.Size())
}

func TestInMemoryCache_Cleanup(t *testing.T) {
	c := NewInMemoryCache(Config{
		MaxEntries:  100,
		TTL:         10 * time.Millisecond,
		CleanupFreq: 20 * time.Millisecond,
	})
	defer c.Stop()

	c.Set(
		&formatter.FormatRequest{Content: "a"},
		&formatter.FormatResult{Content: "a"},
	)
	c.Set(
		&formatter.FormatRequest{Content: "b"},
		&formatter.FormatResult{Content: "b"},
	)
	assert.Equal(t, 2, c.Size())

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 0, c.Size())
}

func TestDefaultCacheConfig(t *testing.T) {
	cfg := DefaultCacheConfig()
	assert.Equal(t, 10000, cfg.MaxEntries)
	assert.Equal(t, 1*time.Hour, cfg.TTL)
	assert.Equal(t, 5*time.Minute, cfg.CleanupFreq)
}

func TestFormatCacheInterface(t *testing.T) {
	// Verify InMemoryCache implements FormatCache interface.
	var _ FormatCache = (*InMemoryCache)(nil)
}
