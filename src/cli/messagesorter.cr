require "option_parser"
require "../open101"

output_dir = ""
OptionParser.parse do |parser|
  parser.banner = "Usage: messagesorter [options] <capture-file>"
  parser.on("--out=DIR", "Output directory") { |value| output_dir = value }
end

if ARGV.size != 1
  STDERR.puts "capture-file is required"
  exit 2
end

path, result = Open101::MessageSorter::Sorter.process_file(ARGV[0], output_dir.empty? ? nil : output_dir)
puts "wrote #{result.messages.size} messages for service #{result.service_name} (#{result.service_id}) to #{path}"
