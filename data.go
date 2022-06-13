package main

type UserData struct {
	Id   int
	Name string
	Age  int
	Born string
}

type LinkData struct {
	Short, Long string
}

type ManagePageData struct {
	User  UserData
	Links []LinkData
}
