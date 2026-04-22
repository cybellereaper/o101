module Open101
  module WizServer
    struct Character
      getter id : UInt32
      getter account_id : UInt32
      property name : String

      def initialize(@id : UInt32, @account_id : UInt32, @name : String)
      end
    end
  end
end
