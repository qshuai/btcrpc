package main

func dispatch() {
	spendableCount := len(input)
	listunspentLimit := conf.DefaultInt("exec::listunspent_limit", DefaultListunspentLimit)

	iteration := spendableCount - listunspentLimit
	if iteration < 0 {
		count := conf.DefaultInt("tx::output_limit", OutputLimit)
		iteration = iteration / count * 2

		for i := 0; i < iteration; i++ {
			s2mTx(true)
		}
	}

	if len(lessCoin) > 5000 {
		m2sTx(true)
	}

	for {
		s2sTx(false)
	}
}
