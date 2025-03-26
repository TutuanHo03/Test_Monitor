package main
type EmulatorApi interface {
  ListUes() []string
  ListGnbs() []string
  AddUe(supi string, triggerRegister bool) bool
}

type UeApi interface {
  Register(isEmergency bool)
  Deregister(deregisterType uint8) bool
  CreateSession(slice string, dnName string, sessionType uint8) bool
}

type GnbApi interface {
  ReleaseUe(ueId string) bool
  ReleaseSession(ueId string, sessionId uint8) bool
}

func createUeApi() UeApi {
  ...
}

func createEmulatorApi() EmulatorApi {
  ...
}

func createGnbApi() GnbApi {
  ...
}

func addCommands(..) {
  eAPi := createEmulatorApi()
  ueApi := createUeAPi()
  gnbApi := createGnbApi()


  //define commands

  {
    Name:
    Arguments:
    Func: func(ctx ishell.Context) {
      //extract agurment here
      ueId := ...
      //then call API
      gnbApi.ReleaseUe(ueId)
    }
  }
}
