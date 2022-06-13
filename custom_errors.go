package main

type NoSuchStatementError struct{}

func (e *NoSuchStatementError) Error() string {
	return "No such statement declared"
}

type Unauthorized struct{}

func (e *Unauthorized) Error() string {
	return "Unauthorized access"
}

type BadRequest struct{}

func (e *BadRequest) Error() string {
	return "Bad request"
}

type NoSuchUser struct{}

func (e *NoSuchUser) Error() string {
	return "User does not exist"
}

type InvalidInput struct{}

func (e *InvalidInput) Error() string {
	return "Input was unvalid"
}
