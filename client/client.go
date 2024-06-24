package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

type Bid struct {
	Bid string `json:"bid"`
}

func main() {

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/cotacao", nil)
	if err != nil {
		panic(err)
	}

	res, err := http.DefaultClient.Do(req)
	if err, ok := err.(net.Error); ok && err.Timeout() {
		fmt.Printf("timeout na chamada da requisicao.\n")
	}

	if err != nil {
		panic(err)
	}

	defer res.Body.Close()

	resposta, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao ler a resposta: %v \n", err)
	}

	fmt.Println(string(resposta))

	var bidresp Bid
	err = json.Unmarshal(resposta, &bidresp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao fazer o parse da resposta: %v \n", err)
	}

	file, err := os.Create("cotacao.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao criar arquivo: %v \n", err)
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("Dolar: %s", bidresp.Bid))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao gravar arquivo: %v \n", err)
	}

}
