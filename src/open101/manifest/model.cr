module Open101
  module Manifest
    struct FileEntry
      include JSON::Serializable

      @[JSON::Field(key: "src")]
      getter source : String
      @[JSON::Field(key: "dst")]
      getter target : String
      getter size : Int64
      getter sha256 : String
      getter mode : String?

      def initialize(@source : String, @target : String, @size : Int64, @sha256 : String, @mode : String? = nil)
      end
    end

    struct Document
      include JSON::Serializable
      getter version : String
      getter files : Array(FileEntry)

      def initialize(@version : String, @files : Array(FileEntry))
      end
    end

    class ValidationError < Exception; end

    class Parser
      def self.parse(raw : String) : Document
        parsed = Document.from_json(raw)
        validate!(parsed)
        parsed
      rescue ex : JSON::SerializableError
        raise ValidationError.new("manifest decode failed: #{ex.message}")
      end

      private def self.validate!(manifest : Document)
        raise ValidationError.new("version is required") if manifest.version.strip.empty?
        raise ValidationError.new("files cannot be empty") if manifest.files.empty?

        manifest.files.each_with_index do |entry, index|
          validate_entry!(entry, index)
        end
      end

      private def self.validate_entry!(entry : FileEntry, index : Int32)
        raise ValidationError.new("files[#{index}].src is required") if entry.source.strip.empty?
        raise ValidationError.new("files[#{index}].dst is required") if entry.target.strip.empty?
        raise ValidationError.new("files[#{index}].size must be positive") if entry.size <= 0
        raise ValidationError.new("files[#{index}].sha256 must be 64 hex chars") unless entry.sha256.matches?(/\A[0-9a-fA-F]{64}\z/)
        raise ValidationError.new("files[#{index}].src must be relative") if entry.source.starts_with?("/")
        raise ValidationError.new("files[#{index}].dst must be relative") if entry.target.starts_with?("/")
        raise ValidationError.new("files[#{index}].dst must use forward slashes") if entry.target.includes?('\\')
      end
    end
  end
end
