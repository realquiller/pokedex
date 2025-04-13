package main

import (
	"fmt"
	"testing"
	"time"
)

func TestCleanInput(t *testing.T) {
	cases := []struct {
		input    string
		expected []string
	}{
		{
			input:    "  hello  world  ",
			expected: []string{"hello", "world"},
		},
		{
			input:    "HALLO thaRe ",
			expected: []string{"hallo", "thare"},
		},
		{
			input:    " 4566        fsdfsf WASDSADsds  ",
			expected: []string{"4566", "fsdfsf", "wasdsadsds"},
		},
		{
			input:    "guASDDDSDASDSDSAD KIGDasdasdadSD dsdsad DADSADSADA hmm >>>",
			expected: []string{"guasdddsdasdsdsad", "kigdasdasdadsd", "dsdsad", "dadsadsada", "hmm", ">>>"},
		},
		{
			input:    "___br O somehoW     ",
			expected: []string{"___br", "o", "somehow"},
		},
		{
			input:    "NAAH brosk67!!!",
			expected: []string{"naah", "brosk67!!!"},
		},
	}

	for _, c := range cases {
		actual := cleanInput(c.input)
		if len(actual) != len(c.expected) {
			t.Errorf("For input %q: Expected %d words, but got %d", c.input, len(c.expected), len(actual))
		}
		for i := range actual {
			word := actual[i]
			expectedWord := c.expected[i]

			if word != expectedWord {
				t.Errorf("For input %q at index %d: Expected %q, but got %q", c.input, i, expectedWord, word)

			}
		}
	}
}

func TestAddGet(t *testing.T) {
	const interval = 5 * time.Second
	cases := []struct {
		key string
		val []byte
	}{
		{
			key: "https://example.com",
			val: []byte("testdata"),
		},
		{
			key: "https://example.com/path",
			val: []byte("moretestdata"),
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("Test case %v", i), func(t *testing.T) {
			cache := NewCache(interval)
			cache.Add(c.key, c.val)
			val, ok := cache.Get(c.key)
			if !ok {
				t.Errorf("expected to find key")
				return
			}
			if string(val) != string(c.val) {
				t.Errorf("expected to find value")
				return
			}
		})
	}
}

func TestReapLoop(t *testing.T) {
	const baseTime = 5 * time.Millisecond
	const waitTime = baseTime + 5*time.Millisecond
	cache := NewCache(baseTime)
	cache.Add("https://example.com", []byte("testdata"))

	_, ok := cache.Get("https://example.com")
	if !ok {
		t.Errorf("expected to find key")
		return
	}

	time.Sleep(waitTime)

	_, ok = cache.Get("https://example.com")
	if ok {
		t.Errorf("expected to not find key")
		return
	}
}
