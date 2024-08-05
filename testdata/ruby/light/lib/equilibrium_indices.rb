def equilibrium_indices(sequence)
    # Determine total sum.
    total_sum = 0
    for n in sequence
        total_sum += n
    end

    # Initialize variables for running sum and result string.
    running_sum = 0
    index_list = ""

    # Compare running sum to remaining sum to find equilibrium indices.
    sequence.each_with_index do |n, i|
        if total_sum - running_sum - n == running_sum
            index_list += "#{i};"
        end
        running_sum += n
    end

    return index_list
end
