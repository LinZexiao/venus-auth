package cli

import (
	"fmt"
	"strconv"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/venus-auth/auth"
	"github.com/filecoin-project/venus-auth/core"
	"github.com/filecoin-project/venus-auth/storage"
)

var userSubCommand = &cli.Command{
	Name:  "user",
	Usage: "user command",
	Subcommands: []*cli.Command{
		addUserCmd,
		updateUserCmd,
		listUsersCmd,
		activeUserCmd,
		getUserCmd,
		hasMinerCmd,
		joinRewardPoolCmd,
		rateLimitSubCmds,
	},
}

var addUserCmd = &cli.Command{
	Name:  "add",
	Usage: "add user",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Required: true,
			Usage:    "required",
		},
		&cli.StringFlag{
			Name: "miner",
		},
		&cli.StringFlag{
			Name: "comment",
		},
		&cli.IntFlag{
			Name:  "sourceType",
			Value: 0,
		},
		&cli.IntFlag{
			Name:  "rewardPoolState",
			Value: core.NotJoin,
			Usage: "Status of users in the reward pool, 0: not join, 1: joined, 2: exited",
		},
	},
	Action: func(ctx *cli.Context) error {
		client, err := GetCli(ctx)
		if err != nil {
			return err
		}
		name := ctx.String("name")
		comment := ctx.String("comment")
		sourceType := ctx.Int("sourceType")
		user := &auth.CreateUserRequest{
			Name:       name,
			Comment:    comment,
			State:      0,
			SourceType: sourceType,
		}
		if ctx.IsSet("miner") {
			mAddr, err := address.NewFromString(ctx.String("miner"))
			if err != nil {
				return err
			}
			user.Miner = mAddr.String()

			has, err := client.HasMiner(&auth.HasMinerRequest{Miner: user.Miner})
			if err != nil {
				return err
			}
			if has {
				return xerrors.Errorf("miner %s exist", user.Miner)
			}
		}
		if ctx.IsSet("rewardPoolState") {
			user.RewardPoolState = ctx.Int("rewardPoolState")
			if user.RewardPoolState != core.NotJoin && user.RewardPoolState != core.Joined {
				return xerrors.Errorf("unexpected state: %d, please choice %d or %d", user.RewardPoolState, core.NotJoin, core.Joined)
			}
			if user.RewardPoolState == core.Joined {
				if len(user.Miner) == 0 {
					return xerrors.New("want to join reward pool but not set miner")
				}
			}
		}
		res, err := client.CreateUser(user)
		if err != nil {
			return err
		}
		fmt.Printf("add user success: %s\n", res.Id)
		return nil
	},
}

var updateUserCmd = &cli.Command{
	Name:  "update",
	Usage: "update user",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Required: true,
		},
		&cli.StringFlag{
			Name: "miner",
		},
		&cli.StringFlag{
			Name: "comment",
		},
		&cli.IntFlag{
			Name: "sourceType",
		},
		&cli.IntFlag{
			Name: "state",
		},
	},
	Action: func(ctx *cli.Context) error {
		client, err := GetCli(ctx)
		if err != nil {
			return err
		}
		req := &auth.UpdateUserRequest{
			Name: ctx.String("name"),
		}
		if ctx.IsSet("miner") {
			addr, err := address.NewFromString(ctx.String("miner"))
			if err != nil {
				return err
			}
			req.Miner = addr.String()
			req.KeySum |= 1

			has, err := client.HasMiner(&auth.HasMinerRequest{Miner: req.Miner})
			if err != nil {
				return err
			}
			if has {
				return xerrors.Errorf("miner %s exist", req.Miner)
			}
		}
		if ctx.IsSet("comment") {
			req.Comment = ctx.String("comment")
			req.KeySum |= 2
		}
		if ctx.IsSet("state") {
			req.State = ctx.Int("state")
			req.KeySum |= 4
		}
		if ctx.IsSet("sourceType") {
			req.SourceType = ctx.Int("sourceType")
			req.KeySum |= 8
		}
		err = client.UpdateUser(req)
		if err != nil {
			return err
		}
		fmt.Println("update user success")
		return nil
	},
}

var activeUserCmd = &cli.Command{
	Name:      "active",
	Usage:     "update user",
	Flags:     []cli.Flag{},
	ArgsUsage: "name",
	Action: func(ctx *cli.Context) error {
		client, err := GetCli(ctx)
		if err != nil {
			return err
		}

		if ctx.NArg() != 1 {
			return xerrors.New("expect name")
		}

		req := &auth.UpdateUserRequest{
			Name: ctx.Args().Get(0),
		}

		req.State = 1
		req.KeySum += 4

		err = client.UpdateUser(req)
		if err != nil {
			return err
		}
		fmt.Println("active user success")
		return nil
	},
}

var joinRewardPoolCmd = &cli.Command{
	Name:      "join-reward-pool",
	Usage:     "user join reward pool",
	Flags:     []cli.Flag{},
	ArgsUsage: "name",
	Action: func(ctx *cli.Context) error {
		client, err := GetCli(ctx)
		if err != nil {
			return err
		}

		if ctx.NArg() != 1 {
			return xerrors.New("expect name")
		}

		req := &auth.UpdateUserRequest{
			Name: ctx.Args().Get(0),
		}

		user, err := client.GetUser(&auth.GetUserRequest{Name: ctx.Args().Get(0)})
		if err != nil {
			return err
		}
		if user.Miner == address.Undef {
			return xerrors.New("miner is empty")
		}
		if user.RewardPoolState == core.Joined {
			return xerrors.New("miner already join reward pool")
		}

		req.RewardPoolState = core.Joined
		req.KeySum += 16
		req.JoinRewardPoolTime = time.Now().Unix()
		req.KeySum += 32

		err = client.UpdateUser(req)
		if err != nil {
			return err
		}
		fmt.Println("join reward pool success")
		return nil
	},
}

var listUsersCmd = &cli.Command{
	Name:  "list",
	Usage: "list users",
	Flags: []cli.Flag{
		&cli.UintFlag{
			Name:  "skip",
			Value: 0,
		},
		&cli.UintFlag{
			Name:  "limit",
			Value: 20,
		},
		&cli.IntFlag{
			Name:  "state",
			Value: 0,
		},
		&cli.IntFlag{
			Name:  "sourceType",
			Value: 0,
		},
		&cli.IntFlag{
			Name:  "rewardPoolState",
			Value: core.NotJoin,
		},
	},
	Action: func(ctx *cli.Context) error {
		client, err := GetCli(ctx)
		if err != nil {
			return err
		}
		req := &auth.ListUsersRequest{
			Page: &core.Page{
				Limit: ctx.Int64("limit"),
				Skip:  ctx.Int64("skip"),
			},
			SourceType:      ctx.Int("sourceType"),
			State:           ctx.Int("state"),
			RewardPoolState: ctx.Int("rewardPoolState"),
		}
		if ctx.IsSet("sourceType") {
			req.KeySum += 1
		}
		if ctx.IsSet("state") {
			req.KeySum += 2
		}
		if ctx.IsSet("rewardPoolState") {
			req.KeySum += 4
		}
		users, err := client.ListUsers(req)
		if err != nil {
			return err
		}

		for k, v := range users {
			fmt.Println("number:", k+1)
			fmt.Println("name:", v.Name)
			fmt.Println("miner:", v.Miner)
			fmt.Println("sourceType:", v.SourceType, "\t// miner:1")
			fmt.Println("state", v.State, "\t// 0: disable, 1: enable")
			fmt.Println("comment:", v.Comment)
			fmt.Println("rewardPoolState:", v.RewardPoolState, "\t// 0: not join, 1: joined, 2: exited")
			if v.RewardPoolState != core.NotJoin {
				fmt.Println("joinRewardPoolTime:", time.Unix(v.JoinRewardPoolTime, 0).Format(time.RFC1123))
				if v.RewardPoolState == core.Exited {
					fmt.Println("exitRewardPoolTime:", time.Unix(v.ExitRewardPoolTime, 0).Format(time.RFC1123))
				}
			}
			fmt.Println("createTime:", time.Unix(v.CreateTime, 0).Format(time.RFC1123))
			fmt.Println("updateTime:", time.Unix(v.CreateTime, 0).Format(time.RFC1123))
			fmt.Println()
		}
		return nil
	},
}

var getUserCmd = &cli.Command{
	Name:      "get",
	Usage:     "get user by name",
	ArgsUsage: "<name>",
	Action: func(ctx *cli.Context) error {
		client, err := GetCli(ctx)
		if err != nil {
			return err
		}
		if ctx.NArg() != 1 {
			return xerrors.Errorf("specify user name")
		}
		name := ctx.Args().Get(0)
		user, err := client.GetUser(&auth.GetUserRequest{Name: name})
		if err != nil {
			return err
		}

		fmt.Println("name:", user.Name)
		fmt.Println("miner:", user.Miner)
		fmt.Println("sourceType:", user.SourceType, "\t// miner:1")
		fmt.Println("state", user.State, "\t// 0: disable, 1: enable")
		fmt.Println("comment:", user.Comment)
		fmt.Println("rewardPoolState:", user.RewardPoolState, "\t// 0: not join, 1: joined, 2: exited")
		if user.RewardPoolState != core.NotJoin {
			fmt.Println("joinRewardPoolTime:", time.Unix(user.JoinRewardPoolTime, 0).Format(time.RFC1123))
			if user.RewardPoolState == core.Exited {
				fmt.Println("exitRewardPoolTime:", time.Unix(user.ExitRewardPoolTime, 0).Format(time.RFC1123))
			}
		}
		fmt.Println("createTime:", time.Unix(user.CreateTime, 0).Format(time.RFC1123))
		fmt.Println("updateTime:", time.Unix(user.CreateTime, 0).Format(time.RFC1123))
		fmt.Println()
		return nil
	},
}

var hasMinerCmd = &cli.Command{
	Name:      "has",
	Usage:     "check miner exit",
	ArgsUsage: "<miner>",
	Action: func(ctx *cli.Context) error {
		client, err := GetCli(ctx)
		if err != nil {
			return err
		}
		if ctx.NArg() != 1 {
			return xerrors.Errorf("specify miner address")
		}
		miner := ctx.Args().Get(0)
		addr, err := address.NewFromString(miner)
		if err != nil {
			return err
		}

		has, err := client.HasMiner(&auth.HasMinerRequest{Miner: addr.String()})
		if err != nil {
			return err
		}
		fmt.Println(has)
		return nil
	},
}

var rateLimitSubCmds = &cli.Command{
	Name: "rate-limit",
	Subcommands: []*cli.Command{
		rateLimitAdd,
		rateLimitUpdate,
		rateLimitGet,
		rateLimitDel},
}

var rateLimitGet = &cli.Command{
	Name:      "get",
	Usage:     "get user request rate limit",
	Flags:     []cli.Flag{},
	ArgsUsage: "name",
	Action: func(ctx *cli.Context) error {
		client, err := GetCli(ctx)
		if err != nil {
			return err
		}

		if ctx.NArg() != 1 {
			return xerrors.New("expect name")
		}

		name := ctx.Args().Get(0)
		var limits []*storage.UserRateLimit
		limits, err = client.GetUserRateLimit(name, "")
		if err != nil {
			return err
		}

		if len(limits) == 0 {
			fmt.Printf("user have no request rate limit\n")
		} else {
			for _, l := range limits {
				fmt.Printf("user:%s, limit id:%s, request limit amount:%d, duration:%.2f(h)\n",
					l.Name, l.Id, l.ReqLimit.Cap, l.ReqLimit.ResetDur.Hours())
			}
		}
		return nil
	},
}

var rateLimitAdd = &cli.Command{
	Name:  "add",
	Usage: "add user request rate limit",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "id", Usage: "rate limit id to update"},
	},
	ArgsUsage: "user rate-limit add <name> <limitAmount> <duration(2h, 1h:20m, 2m10s)>",
	Action: func(ctx *cli.Context) error {
		client, err := GetCli(ctx)
		if err != nil {
			return err
		}

		if ctx.Args().Len() < 3 {
			return cli.ShowAppHelp(ctx)
		}

		name := ctx.Args().Get(0)

		if res, _ := client.GetUserRateLimit(name, ""); len(res) > 0 {
			return fmt.Errorf("user rate limit:%s exists", res[0].Id)
		}

		var limitAmount uint64
		var resetDuration time.Duration
		if limitAmount, err = strconv.ParseUint(ctx.Args().Get(1), 10, 64); err != nil {
			return err
		}
		if resetDuration, err = time.ParseDuration(ctx.Args().Get(2)); err != nil {
			return err
		}
		if resetDuration <= 0 {
			return fmt.Errorf("reset duratoin must be positive")
		}

		userLimit := &auth.UpsertUserRateLimitReq{
			Name:     name,
			ReqLimit: storage.ReqLimit{Cap: int64(limitAmount), ResetDur: resetDuration},
		}

		if ctx.IsSet("id") {
			userLimit.Id = ctx.String("id")
		}

		if userLimit.Id, err = client.UpsertUserRateLimit(userLimit); err != nil {
			return err
		}

		fmt.Printf("upsert user rate limit success:\t%s\n", userLimit.Id)

		return nil
	},
}

var rateLimitUpdate = &cli.Command{
	Name:      "update",
	Usage:     "update user request rate limit",
	Flags:     []cli.Flag{},
	ArgsUsage: "<name> <rate-limit-id> <limitAmount> <duration(2h, 1h:20m, 2m10s)>",
	Action: func(ctx *cli.Context) error {
		client, err := GetCli(ctx)
		if err != nil {
			return err
		}

		if ctx.Args().Len() != 4 {
			return cli.ShowAppHelp(ctx)
		}

		name := ctx.Args().Get(0)
		id := ctx.Args().Get(1)

		if res, err := client.GetUserRateLimit(name, id); err != nil {
			return err
		} else if len(res) == 0 {
			return fmt.Errorf("user rate limit:%s NOT exists", id)
		}

		var limitAmount uint64
		var resetDuration time.Duration
		if limitAmount, err = strconv.ParseUint(ctx.Args().Get(2), 10, 64); err != nil {
			return err
		}
		if resetDuration, err = time.ParseDuration(ctx.Args().Get(3)); err != nil {
			return err
		}
		if resetDuration <= 0 {
			return fmt.Errorf("reset duratoin must be positive")
		}

		userLimit := &auth.UpsertUserRateLimitReq{
			Id: id, Name: name,
			ReqLimit: storage.ReqLimit{Cap: int64(limitAmount), ResetDur: resetDuration},
		}

		if userLimit.Id, err = client.UpsertUserRateLimit(userLimit); err != nil {
			return err
		}

		fmt.Printf("upsert user rate limit success:\t%s\n", userLimit.Id)

		return nil
	},
}

var rateLimitDel = &cli.Command{
	Name:      "del",
	Usage:     "delete user request rate limit",
	Flags:     []cli.Flag{},
	ArgsUsage: "user rate-limit <user> <rate-limit-id> ",
	Action: func(ctx *cli.Context) error {
		client, err := GetCli(ctx)
		if err != nil {
			return err
		}

		if ctx.Args().Len() != 2 {
			return cli.ShowAppHelp(ctx)
		}

		var delReq = &auth.DelUserRateLimitReq{
			Name: ctx.Args().Get(0),
			Id:   ctx.Args().Get(1)}

		if res, err := client.GetUserRateLimit(delReq.Name, delReq.Id); err != nil {
			return err
		} else if len(res) == 0 {
			fmt.Printf("user:%s, rate-limit-id:%s Not exits\n", delReq.Name, delReq.Id)
			return nil
		}

		var id string
		if id, err = client.DelUserRateLimit(delReq); err != nil {
			return err
		}
		fmt.Printf("delete rate limit success, %s\n", id)
		return nil
	},
}
