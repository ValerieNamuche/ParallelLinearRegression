package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

var (
	x []float64 //Datos de entrada "X" para el entrenamiento
	y []float64 //Datos de salida "Y" para el entrenamiento

	data []Persona

	test []float64 //Vector de datos para realizar pruebas del algoritmo

	mx float64 //Media de datos del vector "X"
	my float64 //Media de datos del vector "Y"

	sxy float64 //Suma de las desviaciones cruzadas de “x” y “y”
	sxx float64 //Suma de las desviaciones al cuadrado de “x”

	b0 float64 //Punto de interseccion de la linea de regresión
	b1 float64 //Pendiente de la linea de regresión

	rmse float64 //Error cuadrático medio

	wg1 sync.WaitGroup
	wg2 sync.WaitGroup
	wg3 sync.WaitGroup

	//rango del dataset que va a tomar
	left  int
	rigth int
)

type Persona struct {
	Height float64 `json:"height"` //estatura
	Weight float64 `json:"weight"` //peso
}

func leerDatos() {
	//abrir csv
	csvFile, err := os.Open("Datasets/weights_heights.csv")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Successfully Opened CSV file")
	defer csvFile.Close()
	//recuperar informacion del dataset
	csvLines, err := csv.NewReader(csvFile).ReadAll()
	if err != nil {
		fmt.Println(err)
	}
	//procesar dataset
	for i, line := range csvLines {
		if i >= left && i < rigth {

			weitht, _ := strconv.ParseFloat(line[1], 64)
			height, _ := strconv.ParseFloat(line[2], 64)
			per := Persona{
				Height: height,
				Weight: weitht,
			}
			data = append(data, per)
		}
		//fmt.Println(height, weitht)
	}

}

//Calculo del SXX
func calcularSXX() {
	chi := make(chan int, len(data))
	cho := make(chan float64, len(data))
	var wg sync.WaitGroup
	wg.Add(len(data))

	var ans float64 = 0
	for i := 0; i < len(data); i++ {
		chi <- i
		go func() {
			p := <-chi
			aux := float64(data[p].Height)
			cho <- (aux - mx) * (aux - mx)
			wg.Done()
		}()
		ans += <-cho
	}
	wg.Wait()
	sxx = ans
	wg1.Done()
	//fmt.Println(*S)
}

//Calculo del SXY
func calcularSXY() {
	chi := make(chan int, len(data))
	cho := make(chan float64, len(data))
	var wg sync.WaitGroup
	wg.Add(len(data))

	var ans float64 = 0
	for i := 0; i < len(data); i++ {
		chi <- i
		go func() {
			p := <-chi
			cho <- (data[p].Height - mx) * (data[p].Weight - my)
			wg.Done()
		}()
		ans += <-cho
	}
	sxy = ans
	wg1.Done()
	//fmt.Println(*S)
}

//Cálculo del promedio
func average() {
	var ans1 float64 = 0
	var ans2 float64 = 0
	for i, v := range data {
		if i >= 0 && i < len(data) {
			ans1 += float64(v.Height)
			ans2 += v.Weight
		}
	}
	mx = ans1 / float64(len(data))
	my = ans2 / float64(len(data))

	//fmt.Println(*m)
}

//h(xi)=b0*b1*xi (Calculo del valor previsto)
func prediction(xi float64) float64 {
	return b0 + b1*xi
}

//Cálculo del error cuadrático medio
func calcularRMSE() {
	chi := make(chan int, len(data))
	cho := make(chan float64, len(data))
	var wg sync.WaitGroup
	wg.Add(len(data))

	var ans float64 = 0
	for i := 0; i < len(data); i++ {
		chi <- i
		go func() {
			p := <-chi

			cho <- ((data[p].Weight) - (prediction(data[p].Height))) * ((data[p].Weight) - (prediction(data[p].Height)))
			wg.Done()
		}()
		ans += <-cho
	}
	wg.Wait()
	ans = math.Sqrt(ans / float64(len(data)))
	rmse = ans

	fmt.Println("El error cuadrático medio es de ±", fmt.Sprintf("%.2f", rmse), "kg")
	wg2.Done()
}

//////////////////////////////////
var remotehost string

//envio de datos
func enviar(per Persona) {
	conn, _ := net.Dial("tcp", remotehost)
	defer conn.Close()
	//encriptar datos
	jsonBytes, _ := json.Marshal(per)
	fmt.Fprintf(conn, "%s\n", string(jsonBytes))
}
func enviarError() {
	conn, _ := net.Dial("tcp", remotehost)
	defer conn.Close()
	fmt.Fprintf(conn, "%.2f\n", rmse)
}

//recepción de datos
func manejador(con net.Conn) {
	defer con.Close()
	r := bufio.NewReader(con)

	jsonString, _ := r.ReadString('\n')

	//desencriptar mensaje
	var persona Persona
	json.Unmarshal([]byte(jsonString), &persona)

	//estimar peso con la altura ingresada
	persona.Weight = prediction(float64(persona.Height))

	//mostrar y enviar
	mostrar, _ := json.Marshal(persona)
	fmt.Println("Llegó y se procesó correctamente", string(mostrar))
	enviar(persona)
}
func setRango(con net.Conn) { //setea el rango deld dataset con el cual se va a entrenar
	defer con.Close()
	r := bufio.NewReader(con)
	str, _ := r.ReadString('\n')
	left, _ = strconv.Atoi(strings.TrimSpace(str))
	str, _ = r.ReadString('\n')
	rigth, _ = strconv.Atoi(strings.TrimSpace(str))
	fmt.Println(left, rigth)
	wg3.Done()
}

//ejecución del algoritmo
func ejecutarAlgoritmo() {
	leerDatos()

	average()

	wg1.Add(2)

	go calcularSXX()
	go calcularSXY()

	wg1.Wait()

	b1 = sxy / sxx
	b0 = my - b1*mx

	wg2.Add(1)

	go calcularRMSE()

}

func main() {
	//entrada de puerto
	rIng1 := bufio.NewReader(os.Stdin)
	fmt.Print("Ingrese el puerto de recepción:")
	port, _ := rIng1.ReadString('\n')
	port = strings.TrimSpace(port)
	hostname := fmt.Sprintf("localhost:%s", port)

	fmt.Print("Ingrese el puerto MASTER: ")
	port, _ = rIng1.ReadString('\n')
	port = strings.TrimSpace(port)
	remotehost = fmt.Sprintf("localhost:%s", port)

	ln, _ := net.Listen("tcp", hostname)
	defer ln.Close()

	con, _ := ln.Accept()

	wg3.Add(1)
	setRango(con)
	wg3.Wait()

	ejecutarAlgoritmo()
	wg2.Wait()
	enviarError()
	for {
		con, _ := ln.Accept()
		go manejador(con)
	}

}
