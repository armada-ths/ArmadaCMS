package models

// FairDate represents the schedule and timing information for the fair
type FairDate struct {
    Fair struct {
        Description string   `json:"description"`
        Days        []string `json:"days"`
    } `json:"fair"`
    Ticket struct {
        End *string `json:"end"` // Using pointer to allow for null
    } `json:"ticket"`
    IR struct {
        Start      string `json:"start"`
        End        string `json:"end"`
        Acceptance string `json:"acceptance"`
    } `json:"ir"`
    FR struct {
        Start string `json:"start"`
        End   string `json:"end"`
    } `json:"fr"`
    Events struct {
        Start string `json:"start"`
        End   string `json:"end"`
    } `json:"events"`
}

func NewFairDate() FairDate {
    var fairDate FairDate

    // Set fair information
    fairDate.Fair.Description = "THS Armada 2025"
    fairDate.Fair.Days = []string{"2025-11-18", "2025-11-19"}

    // Set ticket information (null end date)
    fairDate.Ticket.End = nil

    // Set IR (Initial Registration) information
    fairDate.IR.Start = "2025-03-03"
    fairDate.IR.End = "2025-05-23"
    fairDate.IR.Acceptance = "2025-06-02"

    // Set FR (Final Registration) information
    fairDate.FR.Start = "2025-08-11"
    fairDate.FR.End = "2025-09-26"

    // Set Events information
    fairDate.Events.Start = "2025-11-03"
    // Handle the empty end date
    emptyEnd := ""
    fairDate.Events.End = emptyEnd

    return fairDate
}