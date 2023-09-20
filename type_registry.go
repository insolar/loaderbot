package loaderbot

var atkRegistry = make(map[string]Attack)

func RegisterAttacker(name string, atk Attack) {
	atkRegistry[name] = atk
}

func AttackerFromString(name string) Attack {
	return atkRegistry[name]
}
