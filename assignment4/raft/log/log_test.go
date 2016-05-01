package log

import (
	"fmt"
	"os"
	"testing"
)

type SubEntry struct {
	Bar int
	Baz string
}

type TestLogEntry struct {
	Str string
	Foo SubEntry
}

func TestMain(m *testing.M) {
	cleanup()
	m.Run()
	cleanup()
}

func TestAppend(t *testing.T) {
	append(t, 0)
	append(t, 50)
}

// Append 50 TestLogEntry instances
func append(t *testing.T, start int) {
	lg := mkLog(t)
	lg.RegisterSampleEntry(TestLogEntry{})

	defer lg.Close()

	for i := start; i < start+50; i++ {
		e := mkSampleEntry(i)
		err := lg.Append(e)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func mkSampleEntry(i int) TestLogEntry {
	return TestLogEntry{
		Str: fmt.Sprintf("log item i = %d", i),
		Foo: SubEntry{
			Bar: i,
			Baz: "baz",
		},
	}
}

// Depends on results of TestAppend
func TestGet(t *testing.T) {
	lg := mkLog(t)
	defer lg.Close()

	checkIndex(t, lg, 99)
	for i := 0; i < 100; i++ {
		checkGet(t, lg, i)
	}
	lg.Close()
}

func checkGet(t *testing.T, lg *Log, i int) {
	data, err := lg.Get(int64(i))
	if err != nil {
		t.Fatal(err)
	}

	expected := mkSampleEntry(i)

	got, ok := data.(TestLogEntry)
	if !ok {
		t.Fatalf("Expected a TestLogEntry, got: %v", data)
	}
	if expected != got {
		t.Fatalf("Expected '%v', got '%v'", expected, got)
	}
}

// Depends on TestAppend, which should have inserted 100 records
func TestTruncate(t *testing.T) {
	lg := mkLog(t)

	// visit some number of records to make sure they are in
	// the LRU
	for i := int64(20); i < 80; i++ {
		lg.Get(i)
	}

	err := lg.TruncateToEnd(70)

	// In the first round, the LRU cache is warm. In the second round,
	// the cache is cold, and ensures that the Get returns what's on disk
	for rounds := 1; rounds <= 2; rounds++ {
		checkIndex(t, lg, 69)
		for i := int64(70); i < 100; i++ {
			_, err = lg.Get(70)
			if err == nil {
				t.Fatal("Expected records to have been purged")
			}
		}

		lg.Close()
		lg = mkLog(t) // cold start
	}
}

func checkIndex(t *testing.T, lg *Log, expected int) {
	i := lg.GetLastIndex()
	if i != int64(expected) {
		t.Fatal("Expected last index to be ", expected, " got ", i)
	}
}

func mkLog(t *testing.T) *Log {
	lg, err := Open("./logtest")
	if err != nil {
		t.Fatal(err)
	}
	lg.SetCacheSize(50) //
	return lg
}

func cleanup() {
	os.RemoveAll("./logtest")
}
