def parse_csv_string(csv_string)
	CSV.parse(csv_string, headers: true)
end
