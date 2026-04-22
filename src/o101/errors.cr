module O101
  class Error < Exception; end

  class ValidationError < Error; end

  class ParseError < Error; end
end
