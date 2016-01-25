package main


import "fmt"
import "bufio"
import "os"



func main() {

     // Create a channel to synchronize goroutines
     
done := make(chan bool)

     // Execute println in goroutine
 fmt.Println("goroutine message")
  //go sss()
     go s(done)
     //fmt.Println( <-done )
       
     reader := bufio.NewReader(os.Stdin)
          ch, _ := reader.ReadString('\n')
          fmt.Println(ch)
}

func s(done chan bool) {
          fmt.Println("goroutine message")
          go ss(done)
          xx := <-done
          fmt.Println(xx)
          fmt.Println("goroutine message")
         
     }

func ss(done chan bool) {

      fmt.Println("main function message")  
      done <- true
      return
}


func sss() {
  for{
fmt.Print(".")
}}




package main

func main() {
    funcChan := make(chan func(), 2)
    resChan := make(chan string, 2)
    funcChan <- func() {
        resChan <- "hi"
    }
    
    funcChan <- func() {
        resChan <- "hello"
    }
    go func() {
        for f := range funcChan {
            f()
        }
    }()
    println(<-resChan)
    println(<-resChan)
}


