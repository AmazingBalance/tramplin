package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"tramplin/backend/internal/config"
	"tramplin/backend/internal/devseed"
	postgresplatform "tramplin/backend/internal/platform/postgres"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := postgresplatform.Open(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	summary, err := devseed.Seed(ctx, db, cfg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(os.Stdout, "Seeded backend data successfully.\n\n")
	fmt.Fprintf(os.Stdout, "Shared password: %s\n", summary.Password)
	fmt.Fprintf(os.Stdout, "Admin curator: %s\n", summary.AdminCuratorEmail)
	fmt.Fprintf(os.Stdout, "Curator: %s\n", summary.CuratorReviewerEmail)
	fmt.Fprintf(os.Stdout, "Employer owner: %s\n", summary.EmployerOwnerEmail)
	fmt.Fprintf(os.Stdout, "Employer recruiter: %s\n", summary.EmployerRecruiterEmail)
	fmt.Fprintf(os.Stdout, "Applicants: %s, %s, %s\n", summary.ApplicantEmails[0], summary.ApplicantEmails[1], summary.ApplicantEmails[2])
	fmt.Fprintf(os.Stdout, "Company slug: %s\n", summary.CompanySlug)
	fmt.Fprintf(os.Stdout, "Public opportunity slug: %s\n", summary.PublicOpportunitySlug)
	fmt.Fprintf(os.Stdout, "Event opportunity slug: %s\n", summary.EventOpportunitySlug)
	fmt.Fprintf(os.Stdout, "Draft opportunity slug: %s\n", summary.DraftOpportunitySlug)
	if summary.MediaSeeded {
		fmt.Fprintf(os.Stdout, "Media assets: seeded into configured object storage.\n")
	} else {
		fmt.Fprintf(os.Stdout, "Media assets: skipped because object storage is not configured.\n")
	}
}
