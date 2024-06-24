package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/google/uuid"
)

type Cotacao struct {
	Usdbrl struct {
		Code       string `json:"code"`
		Codein     string `json:"codein"`
		Name       string `json:"name"`
		High       string `json:"high"`
		Low        string `json:"low"`
		VarBid     string `json:"varBid"`
		PctChange  string `json:"pctChange"`
		Bid        string `json:"bid"`
		Ask        string `json:"ask"`
		Timestamp  string `json:"timestamp"`
		CreateDate string `json:"create_date"`
	} `json:"USDBRL"`
}

type Retorno struct {
	Bid string `json:"bid"`
}

//Servidor utilizando o ServerMUX e o context
//Cada request vai ser tratado por um contexto diferente

func main() {

	mux := http.NewServeMux()
	mux.HandleFunc("/cotacao", HomeHandler)
	http.ListenAndServe(":8080", mux)

}

func HomeHandler(w http.ResponseWriter, r *http.Request) {

	ctx := context.Background() //Criar um contexto para trabalhar os timeouts

	ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond) //Timeout da requisicao 200 milisegundos

	defer cancel()

	realizarCotacao(ctx, w) //Realizar a cotacao da requisição
}

func realizarCotacao(ctx context.Context, w http.ResponseWriter) {

	log.Println("Request iniciada")

	//Fazer a requisicao
	requisição, err := http.Get("https://economia.awesomeapi.com.br/json/last/USD-BRL")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao fazer a requisição: %v \n", err)
	}
	defer requisição.Body.Close() //Já preparamos pra quando terminar fechar a conexao

	//Ler o retorno
	resposta, err := io.ReadAll(requisição.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao ler a resposta: %v \n", err)
	}

	//Fazer unmarshal do JSON para trabalhar as informações no meu servidor
	var cotacao Cotacao
	err = json.Unmarshal(resposta, &cotacao)
	if err != nil {
		panic(err)
	}

	//Tratar timeout de 10 milisegundos para persistir os dados e inserir no banco
	select {
	case <-time.After(10 * time.Millisecond):
		persistirDadosBanco(&cotacao, ctx)
	case <-ctx.Done():
		fmt.Println("Cotacao cancelada por tempo de requisição")
	}

	ret := Retorno{Bid: cotacao.Usdbrl.Bid}
	retres, err := json.Marshal(ret)
	if err != nil {
		panic(err)
	}

	//Retornar resposta com Bid
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(retres)

	log.Println("Request finalizada")
}

func persistirDadosBanco(cotacao *Cotacao, ctx context.Context) {

	db := ConectarCotacao()

	err := InsertCotacao(db, cotacao)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao inserir a cotacao no banco de dados: %v \n", err)
	}

}

func ConectarCotacao() (db *sql.DB) {

	database := "BaseCotacao.db"

	db, err := sql.Open("sqlite3", database)

	if err != nil {
		panic(err)
	}

	return db

}

func InsertCotacao(db *sql.DB, cotacao *Cotacao) error {
	stmt, err := db.Prepare("INSERT INTO DadosCotacao(IdCotacao, Code, Codein, Name, High, Low, VarBid, PctChange, Bid, Ask, Timestamp, CreateDate) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")

	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(uuid.New().String(), cotacao.Usdbrl.Code, cotacao.Usdbrl.Codein, cotacao.Usdbrl.Name, cotacao.Usdbrl.High, cotacao.Usdbrl.Low, cotacao.Usdbrl.VarBid, cotacao.Usdbrl.PctChange, cotacao.Usdbrl.Bid, cotacao.Usdbrl.Ask, cotacao.Usdbrl.Timestamp, cotacao.Usdbrl.CreateDate)

	if err != nil {
		return err
	}

	defer db.Close()

	return nil
}
