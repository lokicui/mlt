package trie

import (
    "os"
    "log"
	"encoding/gob"
	"fmt"
)

type (
	Trie struct {
		root *Node
		size int
	}
	Node struct {
		key       interface{}
		value     interface{}
        parent    *Node
		next      map[string]*Node
		accessCnt int
		hitCnt    int
	}
	iterator struct {
		step int
		node *Node
		prev *iterator
	}
)

func New() *Trie {
	return &Trie{nil, 0}
}

func (p *Node) GetAccessCnt() int {
	return p.accessCnt
}

func (p *Node) GetHitCnt() int {
	return p.hitCnt
}

func (p *Node) GetValue() interface{} {
	return p.value
}

func (p *Node) GetKey() interface{} {
	return p.key
}

func (p *Node) GetChildrenSize() int {
    return len(p.next)
}

func (p *Node) GetParent() (n *Node) {
    return p.parent
}

func (this *Trie) Walk(handler func(p *Node) bool) {
	if this.root != nil {
		this.root.walk(handler)
	}
}

//longest search
func (this *Trie) Search(keyPieces []string) (longest *Node) {
	if this.root == nil {
		return nil
	}
	cur := this.root
	for _, k := range keyPieces {
		if cur.next[k] != nil {
			cur = cur.next[k]
		} else {
            break
		}
        if cur.GetValue() != nil {
            longest = cur
        }
	}
	return longest
}

func (this *Trie) Get(keyPieces []string) *Node {
	if this.root == nil {
		return nil
	}
	cur := this.root
	for _, k := range keyPieces {
		if cur.next[k] != nil {
			cur = cur.next[k]
		} else {
			return nil
		}
	}
	return cur
}

func (this *Trie) Has(keyPieces []string) bool {
	return this.Get(keyPieces).GetValue() != nil
}

func (this *Trie) Init() {
	this.root = nil
	this.size = 0
}
func (this *Trie) Insert(keyPieces []string, value interface{}) {
	if this.root == nil {
		this.root = newNode()
        this.root.parent = this.root
	}

	cur := this.root
	for _, k := range keyPieces {
		cur.accessCnt += 1
        children, ok := cur.next[k]
		if ok {
			cur = children
		} else {
            p := newNode()
            p.parent = cur
            cur.next[k] = p
			cur = p
		}
	}
	if cur.key == nil {
		this.size++
	}
	cur.key = keyPieces
	cur.value = value
	cur.hitCnt += 1
	cur.accessCnt += 1
}
func (this *Trie) Len() int {
	return this.size
}
func (this *Trie) Remove(keyPieces []string) interface{} {
	if this.root == nil || this.Has(keyPieces) == false {
		return nil
	}
	cur := this.root
	for _, k := range keyPieces {
        cur.accessCnt -= 1
        p, ok := cur.next[k]
		if ok {
            if p.accessCnt == 1 {
                delete(cur.next, k)
            }
			cur = p
		} else {
            log.Fatal(nil)
			return nil
		}
	}
	// TODO: cleanup dead Nodes

	val := cur.value

	if cur.value != nil {
		this.size--
		cur.value = nil
		cur.key = nil
	}
	return val
}
func (this *Trie) String() string {
	str := "{"
	i := 0
	this.Walk(func(p *Node) bool {
		if i > 0 {
			str += ", "
		}
		str += fmt.Sprint(p.GetKey(), ":", p.GetValue())
		i++
		return true
	})
	str += "}"
	return str
}

func (this *Trie) Save(fname string) (err error) {
	// Create an encoder and send a value.
	fd, err := os.Create(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer fd.Close()
	enc := gob.NewEncoder(fd)
	err = enc.Encode(this)
	if err != nil {
		log.Fatal("encode:", err)
	}
    return
}

func (this *Trie) Load(fname string) (err error) {
	// Create a decoder and receive a value.
	fd, err := os.Open(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer fd.Close()
	dec := gob.NewDecoder(fd)
	err = dec.Decode(this)
	if err != nil {
		log.Fatal("decode:", err)
	}
    return
}

func newNode() *Node {
	self := &Node{nil, nil, nil, nil, 0, 0}
	self.next = make(map[string]*Node)
	return self
}
func (this *Node) walk(handler func(p *Node) bool) bool {
    for _, v := range this.next {
        if v.key != nil {
            if !handler(v) {
                return false
            }
        }
        if !v.walk(handler) {
            return false
        }
    }
	return true
}
