package engine

import (
	"testing"

	"github.com/keybase/go/libkb"
	keybase_1 "github.com/keybase/protocol/go"
)

func TestLogin(t *testing.T) {
	tc := libkb.SetupTest(t, "login")
	defer tc.Cleanup()

	u1 := CreateAndSignupFakeUser(t, "login")
	G.LoginState.Logout()
	u2 := CreateAndSignupFakeUser(t, "login")
	G.LoginState.Logout()
	u1.LoginOrBust(t)
	G.LoginState.Logout()
	u2.LoginOrBust(t)

	return
}

func createFakeUserWithNoKeys(t *testing.T) (username, passphrase string) {
	username, email := fakeUser(t, "login")
	passphrase = fakePassphrase(t)

	s := NewSignupEngine(G.UI.GetLogUI(), nil, nil)

	// going to just run the join step of signup engine
	if err := s.genTSPassKey(passphrase); err != nil {
		t.Fatal(err)
	}

	if err := s.join(username, email, "202020202020202020202020"); err != nil {
		t.Fatal(err)
	}

	return username, passphrase
}

func TestLoginFakeUserNoKeys(t *testing.T) {
	tc := libkb.SetupTest(t, "login")
	defer tc.Cleanup()

	createFakeUserWithNoKeys(t)

	me, err := libkb.LoadMe(libkb.LoadUserArg{PublicKeyOptional: true})
	if err != nil {
		t.Fatal(err)
	}

	kf := me.GetKeyFamily()
	if kf == nil {
		t.Fatal("user has a nil key family")
	}
	if kf.GetEldest() != nil {
		t.Fatalf("user has an eldest key, they should have no keys: %s", kf.GetEldest())
	}

	ckf := me.GetComputedKeyFamily()
	if ckf != nil {
		t.Errorf("user has a computed key family.  they shouldn't...")

		active := me.GetComputedKeyFamily().HasActiveKey()
		if active {
			t.Errorf("user has an active key, but they should have no keys")
		}
	}
}

func TestLoginAddsKeys(t *testing.T) {
	tc := libkb.SetupTest(t, "login")
	defer tc.Cleanup()

	username, passphrase := createFakeUserWithNoKeys(t)

	G.LoginState.Logout()

	larg := LoginEngineArg{
		Login: libkb.LoginArg{
			Force:      true,
			Prompt:     false,
			Username:   username,
			Passphrase: passphrase,
			NoUi:       true,
		},
		LogUI:    G.UI.GetLogUI(),
		DoctorUI: &ldocui{},
	}
	li := NewLoginEngine()
	if err := li.Run(larg); err != nil {
		t.Fatal(err)
	}
	if err := G.Session.AssertLoggedIn(); err != nil {
		t.Fatal(err)
	}

	// since this user didn't have any keys, login should have fixed that:
	me, err := libkb.LoadMe(libkb.LoadUserArg{PublicKeyOptional: true})
	if err != nil {
		t.Fatal(err)
	}

	kf := me.GetKeyFamily()
	if kf == nil {
		t.Fatal("user has a nil key family")
	}
	if kf.GetEldest() == nil {
		t.Fatal("user has no eldest key")
	}

	ckf := me.GetComputedKeyFamily()
	if ckf == nil {
		t.Fatalf("user has no computed key family")
	}

	//	ckf.DumpToLog(G.UI.GetLogUI())

	active := ckf.HasActiveKey()
	if !active {
		t.Errorf("user has no active key")
	}

	dsk, err := me.GetDeviceSibkey()
	if err != nil {
		t.Fatal(err)
	}
	if dsk == nil {
		t.Fatal("nil sibkey")
	}
}

func createFakeUserWithDetKey(t *testing.T) (username, passphrase string) {
	username, email := fakeUser(t, "login")
	passphrase = fakePassphrase(t)

	s := NewSignupEngine(G.UI.GetLogUI(), nil, nil)

	if err := s.genTSPassKey(passphrase); err != nil {
		t.Fatal(err)
	}

	if err := s.join(username, email, "202020202020202020202020"); err != nil {
		t.Fatal(err)
	}

	// generate the detkey only, using SelfProof
	eng := NewDetKeyEngine(s.me, nil, s.logUI)
	if err := eng.RunSelfProof(&s.tspkey); err != nil {
		t.Fatal(err)
	}

	return username, passphrase
}

func TestLoginDetKeyOnly(t *testing.T) {
	tc := libkb.SetupTest(t, "login")
	defer tc.Cleanup()

	username, passphrase := createFakeUserWithDetKey(t)

	G.LoginState.Logout()

	larg := LoginEngineArg{
		Login: libkb.LoginArg{
			Force:      true,
			Prompt:     false,
			Username:   username,
			Passphrase: passphrase,
			NoUi:       true,
		},
		LogUI:    G.UI.GetLogUI(),
		DoctorUI: &ldocui{},
	}
	li := NewLoginEngine()
	if err := li.Run(larg); err != nil {
		t.Fatal(err)
	}
	if err := G.Session.AssertLoggedIn(); err != nil {
		t.Fatal(err)
	}

	// since this user didn't have a device key, login should have fixed that:
	me, err := libkb.LoadMe(libkb.LoadUserArg{PublicKeyOptional: true})
	if err != nil {
		t.Fatal(err)
	}

	kf := me.GetKeyFamily()
	if kf == nil {
		t.Fatal("user has a nil key family")
	}
	if kf.GetEldest() == nil {
		t.Fatal("user has no eldest key")
	}

	ckf := me.GetComputedKeyFamily()
	if ckf == nil {
		t.Fatalf("user has no computed key family")
	}

	ckf.DumpToLog(G.UI.GetLogUI())

	active := ckf.HasActiveKey()
	if !active {
		t.Errorf("user has no active key")
	}

	dsk, err := me.GetDeviceSibkey()
	if err != nil {
		t.Fatal(err)
	}
	if dsk == nil {
		t.Fatal("nil sibkey")
	}
}

func TestLoginNewDevice(t *testing.T) {
	tc := libkb.SetupTest(t, "login")
	u1 := CreateAndSignupFakeUser(t, "login")
	G.LoginState.Logout()
	tc.Cleanup()

	// redo SetupTest to get a new home directory...should look like a new device.
	tc2 := libkb.SetupTest(t, "login")
	defer tc2.Cleanup()

	docui := &ldocui{}

	larg := LoginEngineArg{
		Login: libkb.LoginArg{
			Force:      true,
			Prompt:     false,
			Username:   u1.Username,
			Passphrase: u1.Passphrase,
			NoUi:       true,
		},
		LogUI:    G.UI.GetLogUI(),
		DoctorUI: docui,
	}

	before := docui.selectSignerCount

	li := NewLoginEngine()
	/*
		if err := li.LoginAndIdentify(larg); err != nil {
			t.Fatal(err)
		}
		if err := G.Session.AssertLoggedIn(); err != nil {
			t.Fatal(err)
		}
	*/

	if err := li.Run(larg); err != ErrNotYetImplemented {
		t.Fatal(err)
	}

	after := docui.selectSignerCount
	if after-before != 1 {
		t.Errorf("doc ui SelectSigner called %d times, expected 1", after-before)
	}
}

type ldocui struct {
	selectSignerCount int
}

func (l *ldocui) PromptDeviceName(sid int) (string, error) {
	return "my test device", nil
}

func (l *ldocui) SelectSigner(devs []keybase_1.DeviceDescription) (res keybase_1.SelectSignerRes, err error) {
	l.selectSignerCount++
	return
}
