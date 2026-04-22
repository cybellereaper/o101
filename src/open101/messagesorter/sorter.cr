module Open101
  module MessageSorter
    struct Result
      getter service_id : String
      getter service_name : String
      getter messages : Array(String)

      def initialize(@service_id : String, @service_name : String, @messages : Array(String))
      end
    end

    class Sorter
      RECORD_BLOCK = /<RECORD>.*?<\/RECORD>/m

      def self.process(raw : String) : Result
        service_id = extract_tag(raw, %(<ServiceID TYPE="UBYT">), "</ServiceID>")
        service_name = extract_tag(raw, %(<ProtocolType TYPE="STR">), "</ProtocolType>")
        close_index = raw.index("</_ProtocolInfo>") || raise "protocol info close tag not found"
        body = raw[(close_index + "</_ProtocolInfo>".size)..]
        body = body.gsub(RECORD_BLOCK, "")
        messages = collect(body)

        Result.new(service_id, service_name, messages)
      end

      def self.process_file(input_path : String, output_dir : String? = nil) : {String, Result}
        result = process(File.read(input_path))
        out_dir = output_dir || File.dirname(input_path)
        Dir.mkdir_p(out_dir)
        output_path = File.join(out_dir, output_filename(result.service_id, result.service_name))
        write_numbered(output_path, result.messages)
        {output_path, result}
      end

      def self.output_filename(service_id : String, service_name : String) : String
        "#{sanitize(service_id)}_#{sanitize(service_name)}.txt"
      end

      def self.write_numbered(path : String, messages : Array(String))
        File.open(path, "w") do |file|
          messages.each_with_index(1) { |msg, i| file.puts("#{i}: #{msg}") }
        end
      end

      private def self.extract_tag(raw : String, open_tag : String, close_tag : String) : String
        start = raw.index(open_tag) || raise "missing tag #{open_tag}"
        start += open_tag.size
        tail = raw[start..]
        ending = tail.index(close_tag) || raise "missing tag #{close_tag}"
        tail[0, ending]
      end

      private def self.collect(body : String) : Array(String)
        set = Set(String).new
        body.each_line do |line|
          trimmed = line.strip
          next if trimmed.empty? || trimmed.starts_with?("</")
          next unless trimmed.starts_with?("<") && trimmed.ends_with?(">")
          set << trimmed[1..-2]
        end
        set.to_a.sort
      end

      private def self.sanitize(value : String) : String
        sanitized = value.strip.gsub(/[\\\/:*?"<>|]/, "-")
        sanitized.empty? ? "unknown" : sanitized
      end
    end
  end
end
