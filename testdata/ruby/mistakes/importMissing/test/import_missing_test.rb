require 'minitest/autorun'
require_relative '../lib/import_missing'

class TestImportMissing < Minitest::Test
	def test_valid_json
        json_string = '{"key": "value"}'
        expected_result = {"key" => "value"}
        assert_equal expected_result, parse_json(json_string)
    end

    def test_empty_json
        json_string = '{}'
        expected_result = {}
        assert_equal expected_result, parse_json(json_string)
    end

    def test_json_with_numbers
        json_string = '{"number": 123}'
        expected_result = {"number" => 123}
        assert_equal expected_result, parse_json(json_string)
    end

    def test_json_with_nested_objects
        json_string = '{"person": {"name": "Alice", "age": 30}}'
        expected_result = {"person" => {"name" => "Alice", "age" => 30}}
        assert_equal expected_result, parse_json(json_string)
    end

    def test_invalid_json
        json_string = '{"key": "value"'
        assert_raises(JSON::ParserError) { parse_json(json_string) }
    end

    def test_json_with_array
        json_string = '[1, 2, 3, 4]'
        expected_result = [1, 2, 3, 4]
        assert_equal expected_result, parse_json(json_string)
    end
end
