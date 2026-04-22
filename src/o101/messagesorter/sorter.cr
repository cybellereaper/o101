module O101
  module MessageSorter
    struct Result
      getter service_id : String
      getter service_name : String
      getter messages : Array(String)

      def initialize(@service_id : String, @service_name : String, @messages : Array(String))
      end

      def output_filename : String
        "#{service_id}_#{service_name}.txt"
      end

      def to_output : String
        messages.map_with_index { |message, i| "#{i + 1}: #{message}" }.join('\n')
      end
    end

    class Sorter
      SERVICE_PATTERN = /service\s*=\s*"([^"]+)"\s+id\s*=\s*"(\d+)"/i
      MESSAGE_PATTERN = /<message>([^<]+)<\/message>/i

      def sort(input : String) : Result
        service_name, service_id = parse_service!(input)
        messages = input.scan(MESSAGE_PATTERN).map(&.[1].strip).reject(&.empty?).uniq.sort
        Result.new(service_id, service_name, messages)
      end

      private def parse_service!(input : String) : {String, String}
        match = SERVICE_PATTERN.match(input)
        raise ParseError.new("service metadata not found") unless match

        {match[1], match[2]}
      end
    end
  end
end
