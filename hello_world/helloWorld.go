package hello_world

func GetGreeting() string {
	const greeting string = "Hello, World!"
	return greeting
}

func GreetWorld() {
	println(GetGreeting())
}
