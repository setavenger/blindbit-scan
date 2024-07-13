package database

import (
	"log"
	"os"
)

type Serialiser interface {
	Serialise() ([]byte, error)
	DeSerialise([]byte) error
}

func WriteToDB(path string, dataStruct Serialiser) error {
	data, err := dataStruct.Serialise()
	if err != nil {
		log.Println(err)
		return err
	}
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func ReadFromDB(path string, dataStruct Serialiser) error {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Println(err)
		return err
	}

	err = dataStruct.DeSerialise(data)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}
