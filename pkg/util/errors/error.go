package errors

func PanicIfErrorOccur(err error) {
	if err != nil {
		panic(err)
	}
}