module O101
  module Manifest
    struct FileEntry
      include JSON::Serializable

      getter src : String
      getter dst : String
      getter size : Int64
      getter sha256 : String
      @[JSON::Field(converter: O101::Manifest::ModeConverter)]
      getter mode : Int32

      def validate!
        raise ValidationError.new("src cannot be blank") if src.blank?
        raise ValidationError.new("dst cannot be blank") if dst.blank?
        raise ValidationError.new("size must be non-negative") if size < 0
        raise ValidationError.new("sha256 must be 64 hex characters") unless sha256.matches?(/\A[0-9a-fA-F]{64}\z/)
      end
    end

    struct Document
      include JSON::Serializable

      getter version : String
      getter files : Array(FileEntry)

      def validate!
        raise ValidationError.new("version cannot be blank") if version.blank?
        files.each &.validate!
      end
    end

    module ModeConverter
      def self.from_json(pull : JSON::PullParser) : Int32
        value = pull.read_string
        value.to_i(8)
      rescue
        raise ParseError.new("invalid file mode: #{value}")
      end

      def self.to_json(value : Int32, json : JSON::Builder)
        json.string(value.to_s(8).rjust(4, '0'))
      end
    end

    struct PatchInfo
      include JSON::Serializable
      getter version : String
      getter manifest : String
      @[JSON::Field(key: "base_url")]
      getter base_url : String
    end
  end
end
