package cli

type Command struct {
	Name string
	Args []string
}

func Parse(args []string) Command {
	if len(args) == 0 {
		return Command{Name: "status"}
	}

	return Command{
		Name: args[0],
		Args: args[1:],
	}
}
