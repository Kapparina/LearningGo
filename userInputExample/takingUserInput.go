package userInputExample

import (
	"fmt"
	"os"

	"LearningGo/error_handling"
)

const minimumAge int = 10

type User struct {
	name string
	age  int
}

func (u *User) getName() {
	print("Enter your name: ")
	err := error_handling.CatchInputError(fmt.Scan(&u.name))
	print(err)
	fmt.Printf("Hello, %s! Let's get started!\n", u.name)
}

func (u *User) getAge() {
	print("Enter your age: ")
	error_handling.CatchInputError(fmt.Scan(&u.age))
	if u.age < minimumAge {
		println("You are too young to play this game!")
		os.Exit(1)
	} else {
		println("You are old enough to play this game!")
	}
}

func Example() {
	println("This is an example of getting user input.")
	var user User
	user.getName()
	user.getAge()
	println("You are ready to play the game!")
}
