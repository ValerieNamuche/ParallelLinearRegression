package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type Persona struct {
	Height float64 `json:"height"`
	Weight float64 `json:"weight"`
}

var (
	n          int
	remotehost []string
)

//envío de datos
func enviar(per Persona, nodo int) { //Envía la clase persona encriptada
	conn, _ := net.Dial("tcp", remotehost[nodo])
	defer conn.Close()

	//encriptar datos
	jsonBytes, _ := json.Marshal(per)

	fmt.Fprintf(conn, "%s\n", string(jsonBytes))
}

func enviarRango(a, b, nodo int) { //envía el rango que tomará cada nodo
	conn, _ := net.Dial("tcp", remotehost[nodo])
	defer conn.Close()
	fmt.Fprintf(conn, "%d\n", a)
	fmt.Fprintf(conn, "%d\n", b)
}

//captura de datos
func manejador(con net.Conn, ch chan Persona) {

	defer con.Close()
	r := bufio.NewReader(con)
	jsonString, _ := r.ReadString('\n')
	//desencriptado de datos
	var persona Persona
	json.Unmarshal([]byte(jsonString), &persona)
	ch <- persona

}

//calculo del error promedio de los nodos
func calcularError(ch chan float64) {
	var err float64 = 0
	for i := 0; i < n; i++ {
		p := <-ch
		err = err + p
	}
	err = err / float64(n)
	fmt.Printf("El error cuadrático medio obtenido por el algoritmo es de ±%.2fkg\n", err)
}

//recibe el error obtenido en cada nodo (Se utiliza RMSE)
func recibirError(con net.Conn, ch chan float64) {
	defer con.Close()
	r := bufio.NewReader(con)

	_err, _ := r.ReadString('\n')
	_err = strings.TrimSpace(_err)
	err, _ := strconv.ParseFloat(_err, 64)
	ch <- err
}

//Calcula el resultado promedio de los nodos
func procesarResultado(ch chan Persona) {
	var resultados []Persona
	for i := 0; i < n; i++ {
		p := <-ch
		resultados = append(resultados, p)
	}
	var promedio float64 = 0
	for _, i := range resultados {
		promedio += i.Weight
	}
	promedio = promedio / float64(n)
	fmt.Printf("El peso estimado para una persona de %.2fm de estatura es de %.2fkg\n", resultados[0].Height/100, promedio)
}

//Calcula el intervalo de datos con el que entrenará cada nodo
func dividirTrabajo(ldataset int) {
	a := int(ldataset / n)
	for i := 0; i < n; i++ {
		left := i * a
		right := (i + 1) * a
		if i == n-1 {
			right = ldataset - 1
		}
		enviarRango(left, right, i)
	}
}

func main() {
	//lectura del dataset para obtener el tamaño
	chPersona := make(chan Persona, n)
	chError := make(chan float64, n)
	csvFile, err := os.Open("Datasets/weights_heights.csv")
	if err != nil {
		fmt.Println(err)
	}
	defer csvFile.Close()
	csvLines, err := csv.NewReader(csvFile).ReadAll()
	if err != nil {
		fmt.Println(err)
	}

	rIng1 := bufio.NewReader(os.Stdin)

	//lectura de puertos
	fmt.Print("Ingrese el puerto receptor:")
	port, _ := rIng1.ReadString('\n')
	port = strings.TrimSpace(port)
	hostname := fmt.Sprintf("localhost:%s", port)

	fmt.Print("Cantidad de nodos n: ")
	nelementos, _ := rIng1.ReadString('\n')
	nelementos = strings.TrimSpace(nelementos)
	n, _ = strconv.Atoi(nelementos)

	for i := 0; i < n; i++ {
		fmt.Printf("Ingrese el puerto de destino %d: ", i+1)
		port, _ := rIng1.ReadString('\n')
		port = strings.TrimSpace(port)
		remotehost = append(remotehost, fmt.Sprintf("localhost:%s", port))
	}

	dividirTrabajo(len(csvLines))
	ln, _ := net.Listen("tcp", hostname)
	defer ln.Close()

	for i := 0; i < n; i++ {
		con, _ := ln.Accept()
		go recibirError(con, chError)
	}

	calcularError(chError)

	for {
		fmt.Print("Ingrese la estatura a estimar (en cm): ")
		_dato, _ := rIng1.ReadString('\n')
		_dato = strings.TrimSpace(_dato)
		dato, _ := strconv.ParseFloat(_dato, 64)
		p := Persona{
			Height: dato,
			Weight: 0,
		}

		for i := 0; i < n; i++ {
			enviar(p, i)
		}

		for i := 0; i < n; i++ {
			con, _ := ln.Accept()
			go manejador(con, chPersona)
		}

		procesarResultado(chPersona)
	}
}
