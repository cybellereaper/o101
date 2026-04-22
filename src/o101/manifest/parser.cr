module O101
  module Manifest
    class Parser
      def self.parse!(json : String) : Document
        document = Document.from_json(json)
        document.validate!
        document
      rescue error : JSON::ParseException
        raise ParseError.new("manifest JSON parse error: #{error.message}")
      end

      def self.parse_patch_info!(json : String) : PatchInfo
        PatchInfo.from_json(json)
      rescue error : JSON::ParseException
        raise ParseError.new("patch info JSON parse error: #{error.message}")
      end
    end
  end
end
