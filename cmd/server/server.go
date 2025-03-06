package main
import (
	"context"
	"log"
	"time"
	"github.com/pekon/mdnscache/pkg/mdnscache"
)

func main() {
	l, err := NewListener("enp8s0")
	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	msgs := make(chan *Msg)
	l.Listen(ctx, msgs)
	for msg := range msgs {
		if msg.Response {
			for _, a := range msg.Answer {
				log.Printf("answer addr: %v msg: %v", msg.Addr, a)
				// TODO: check that it is a PTR and has correct name
				// Cache for TTL time
			}
			for _, e := range msg.Extra {
				log.Printf("extra addr: %v msg: %v", msg.Addr, e)
			}
		} else {
			for _, q := range msg.Question {
				log.Printf("question  addr: %v msg: %v", msg.Addr, q)
			}
		}

	}
}
