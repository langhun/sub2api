package redpacket

// LegacyMethod identifies a public method that currently belongs to
// service.BalanceTransferService and must move with red-packet routing.
type LegacyMethod struct {
	Source      string
	Replacement string
}

// LegacyExtractionBoundary is the complete public red-packet surface to move
// out of BalanceTransferService. Direct-transfer methods, fee preview, and
// transfer administration remain outside this module. The private helpers
// calculateClaimAmount and generateRedPacketCode move with Create and Claim
// when a real runtime is introduced.
var LegacyExtractionBoundary = []LegacyMethod{
	{Source: "BalanceTransferService.CreateRedPacket", Replacement: "redpacket.Creator.Create"},
	{Source: "BalanceTransferService.ClaimRedPacket", Replacement: "redpacket.Claimer.Claim"},
	{Source: "BalanceTransferService.ExpireRedPackets", Replacement: "redpacket.ExpiryRefunder.RefundExpired"},
	{Source: "BalanceTransferService.GetRedPacketDetail", Replacement: "redpacket.QueryService.Get"},
	{Source: "BalanceTransferService.GetRedPacketDetailForUser", Replacement: "redpacket.QueryService.GetForParticipant"},
	{Source: "BalanceTransferService.GetMyRedPackets", Replacement: "redpacket.QueryService.ListCreatedBy and ListClaimedBy"},
	{Source: "BalanceTransferService.GetAllRedPackets", Replacement: "redpacket.QueryService.ListAll"},
}
