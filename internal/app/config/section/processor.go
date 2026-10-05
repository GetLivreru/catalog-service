package section

type (
	Processor struct {
		// TODO: Добавьте поле WebServer
		WebServer ProcessorWebServer `split_words:"true"`
		// Не забудьте про split_words!
	}

	ProcessorWebServer struct {
		// TODO: Добавьте поле ListenPort (uint32)
		// Порт по умолчанию: 8080
		ListenPort uint32 `split_words:"true" default:"8080"`
	}
)
