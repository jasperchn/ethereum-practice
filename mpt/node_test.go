package mpt

import (
	"bytes"
	"fmt"
	"testing"
)

func TestCommonKeyLength(t *testing.T) {
	a, b := []byte{1,2,3,4}, []byte{1,2}
	c, d := []byte{1,2,3,4}, []byte{1,4,2,3,1,4,1}
	e, f := []byte{}, []byte{1,4,2,3,1,4,1}

	fmt.Println(commonKeyLength(a, b))
	fmt.Println(commonKeyLength(c, d))
	fmt.Println(commonKeyLength(e, f))

}


func TestEqual(t *testing.T) {
	//a, b := []byte{1,2,3,4}, []byte{1,2}
	a, b := []byte{}, []byte{1,2}

	fmt.Println(bytes.Equal(a, b))


	//fmt.Println(append(nil,1,2,3))
}



