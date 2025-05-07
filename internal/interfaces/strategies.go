package interfaces

type Strategy interface {
    // Next возвращает URL следующего бэкенд-сервера
    Next() (string, error)
}