package trie

import (
    "fmt"
    "testing"
)


var (
    tt *Trie = New()
)


func TestInsert(t *testing.T) {
    keyPieces := []string {"a", "b", "c"}
    v := "v1"
    tt.Insert(keyPieces, v)
    tt.Insert(keyPieces, v)
    tt.Insert([]string{"a","b","c","d"}, v)
    tt.Insert([]string{"a","b","c","d"}, v)
    tt.Insert([]string{"a","b","c","d"}, v)
    tt.Insert([]string{"a","b","d"}, v)
    tt.Insert([]string{"a","b","e"}, v)
    tt.Insert([]string{"b","c","d"}, v)
    tt.Insert([]string{"b","c","e"}, v)
    node := tt.Get(keyPieces)
    fmt.Printf("%#v\n", node)
}

func TestString(t *testing.T) {
    fmt.Printf("%#v\n", tt.String())
}

func TestSave(t *testing.T) {
    tt.Save("trie.save")
}

func TestLoad(t *testing.T) {
    tt.Load("trie.save")
}
