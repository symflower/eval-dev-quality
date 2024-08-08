def type_array_multidimensional_array_length(x)
	if x.length == 2
	  if x[0].length == 2
		return 2
	  end

	  return 1
	end

	return 0
end
