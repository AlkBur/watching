package watching

import (
	"testing"
	"time"
	"fmt"
	"os"
)

func TestNew(t *testing.T) {
	c := func(arr []os.FileInfo) {
		fmt.Println("start:", time.Now())
		if len(arr)>0{
			for _, f := range arr {
				fmt.Println("--chages:", f)
			}
		}
	}

	w := New(c)
	w.SetTimeout(3000)
	w.AddWatcher("*.go")
	w.Run()
	//w.Close()

	test_name_file := "test_create.go"

	<- time.After(4 * time.Second)
	f,_:= os.Create(test_name_file)
	f.Close()
	defer os.Remove(test_name_file)
	fmt.Println("Create file")

	<- time.After(10 * time.Second)

	w.Close()
}
