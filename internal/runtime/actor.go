package runtime

// Actor facade and Green-thread style API

// ActorRef is a lightweight reference to an actor in an ActorSystem.
type ActorRef struct {
    System *ActorSystem
    ID     ActorID
}

// Spawn creates a new actor with the given behavior and returns its reference.
// This is a convenience wrapper over CreateActor and represents a green-thread style spawn.
func (as *ActorSystem) Spawn(name string, behavior ActorBehavior, cfg ActorConfig) (*ActorRef, error) {
    actor, err := as.CreateActor(name, UserActor, behavior, cfg)
    if err != nil {
        return nil, err
    }
    return &ActorRef{System: as, ID: actor.ID}, nil
}

// Tell sends a message to the actor (fire-and-forget).
func (ref *ActorRef) Tell(msgType MessageType, payload interface{}) error {
    if ref == nil || ref.System == nil {
        return nil
    }
    return ref.System.SendMessage(0, ref.ID, msgType, payload)
}


