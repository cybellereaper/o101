require "socket"

module Open101
  module WizServer
    module Util
      def self.handle_connection(socket : TCPSocket, service_name : String)
        socket.puts "Welcome to the WizTurtle #{service_name} service!"
        line = socket.gets
        return unless line
        socket.puts "You said: #{line.strip}"
      ensure
        socket.close
      end
    end
  end
end
