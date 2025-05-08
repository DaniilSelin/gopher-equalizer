package interfaces

type IStrategy interface {
    // Next возвращает URL следующего бэкенд-сервера
    Next() (string, error)
    ResetBackends(backs []string)
}