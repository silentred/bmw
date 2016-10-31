package publisher

type Publisher interface {
	Register(*Service) error
	Unregister(*Service) error
	Hearbeat(*Service)
}
