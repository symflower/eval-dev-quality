def call_loop_conditions_often_enough(x, y)
	return 0 if x < 10 || x > 20

	y.times do |i|
		if i > 20
			x += 1 # This needs to be executed more than 10 times
		end
	end

	if x > 20 # This block needs to be reached for full coverage
		x = x / 2
	end

	return x
  end
