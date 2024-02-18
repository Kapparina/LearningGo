package interfaceExample

import "fmt"

type gasEngine struct {
	mpg     uint8
	gallons uint8
	owner
}

func (e gasEngine) milesLeft() uint8 {
	return e.gallons * e.mpg
}

type owner struct {
	name string
}

type electricEngine struct {
	mpkwh uint8
	kwh   uint8
}

func (e electricEngine) milesLeft() uint8 {
	return e.kwh * e.mpkwh
}

type engine interface {
	milesLeft() uint8
}

func canMakeIt(e engine, miles uint8) {
	if e.milesLeft() > miles {
		fmt.Println("You can make it!")
	} else {
		fmt.Println("You can't make it!")
	}
}

func ExampleInterfaceUsage() {
	println("This is the interfaceExample package")
	var myEngine gasEngine = gasEngine{mpg: 30, gallons: 10, owner: owner{name: "John"}}
	fmt.Println(myEngine.mpg, myEngine.gallons, myEngine.owner.name)
	canMakeIt(myEngine, 20)

	var myElectricEngine electricEngine = electricEngine{mpkwh: 4, kwh: 10}
	fmt.Println(myElectricEngine.mpkwh, myElectricEngine.kwh)
	canMakeIt(myElectricEngine, 200)
}
