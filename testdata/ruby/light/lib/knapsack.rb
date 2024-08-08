class Item
    attr_reader :weight, :value

    def initialize(weight, value)
        @weight = weight
        @value = value
    end
end

def maximum_value(maximum_weight, items)
    knapsack = Array.new(items.size + 1)
    for item in 0..items.size
        knapsack[item] = Array.new(maximum_weight + 1)
        for weight in 0..maximum_weight
            knapsack[item][weight] = 0
        end
    end

    for item in 0..items.size
        for weight in 0..maximum_weight
            if item == 0 || weight == 0
                knapsack[item][weight] = 0
            elsif items[item - 1].weight > weight
                knapsack[item][weight] = knapsack[item - 1][weight]
            else
                item_value = items[item - 1].value
                item_weight = items[item - 1].weight
                knapsack[item][weight] = [
                    item_value + knapsack[item - 1][weight - item_weight],
                    knapsack[item - 1][weight]
                ].max
            end
        end
    end

    return knapsack[items.size][maximum_weight]
end
