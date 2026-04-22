require "../o101"

input_path = ""
out_dir = ""

OptionParser.parse do |parser|
  parser.banner = "Usage: messagesorter INPUT [--out DIR]"
  parser.on("--out DIR", "Output directory") { |value| out_dir = value }
  parser.unknown_args { |args| input_path = args.first? || "" }
end

abort("input file is required") if input_path.blank?
text = File.read(input_path)
result = O101::MessageSorter::Sorter.new.sort(text)
out_dir = File.dirname(input_path) if out_dir.blank?
out_path = File.join(out_dir, result.output_filename)
File.write(out_path, result.to_output)
puts "wrote #{result.messages.size} messages for service #{result.service_name} (#{result.service_id}) to #{out_path}"
