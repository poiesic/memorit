package main

import (
	"bufio"
	"context"
	"flag"
	"iter"
	"log/slog"
	"os"

	"github.com/poiesic/memorit"
	"github.com/poiesic/memorit/core"
	"github.com/poiesic/memorit/ingestion"
)

var sentences = []string{
	"The quick brown fox jumps over the lazy dog.",
	"A gentle breeze rustled the leaves of the old oak tree.",
	"She found a hidden key in the dusty attic.",
	"The city skyline glowed under the starry night sky.",
	"He whispered secrets to the wind, hoping they would travel far.",
	"Rain drummed on the rooftop, creating a soothing rhythm.",
	"A bright comet streaked across the horizon at midnight.",
	"They laughed together as fireworks painted the evening air.",
	"The ancient library held stories that never faded.",
	"Beneath the waves, coral gardens shimmered in colors unseen.",
	"The hummingbird hovered beside a vibrant purple flower.",
	"A mysterious map led them to a forgotten treasure.",
	"Her heart raced as she stepped onto the stage for the first time.",
	"Sunlight filtered through curtains, turning dust motes into golden specks.",
	"They tasted the sweetest strawberries from the farmer's garden.",
	"The old clock chimed thirteen times in an abandoned town.",
	"A sudden thunderclap shattered the silence of the forest.",
	"He composed a melody that echoed through the valleys.",
	"The desert dunes shifted silently under a pale moon.",
	"A small kitten meowed softly, waiting for warmth.",
	"She painted the sunset with bold strokes of crimson and gold.",
	"A silver fox slipped past the fences into the twilight.",
	"They discovered an ancient rune carved deep within the stone.",
	"The wind carried scents of jasmine from distant gardens.",
	"He built a wooden bridge across the swift river.",
	"Her laughter echoed through the empty halls of the old manor.",
	"A lone wolf howled, echoing into the vast night.",
	"They tasted coffee brewed fresh in the quiet dawn.",
	"The moon rose slowly, casting silver light on the lake.",
	"A child drew a rainbow with crayons on the sidewalk.",
	"He felt the rough bark of the tree against his palm.",
	"She carried a bouquet of wildflowers from the meadow.",
	"The train rattled through tunnels carved into stone.",
	"They watched a parade of balloons float over the town square.",
	"A gentle snowfall blanketed the city in quiet white.",
	"He whispered to the stars, hoping they would hear.",
	"The river's current carried leaves downstream like paper boats.",
	"She hummed a tune she learned from her grandmother.",
	"They explored caves filled with stalactites glittering like chandeliers.",
	"A rustling in the bushes signaled the arrival of deer.",
	"He measured the distance between two distant mountains.",
	"The lighthouse beam cut through fog, guiding sailors safely.",
	"She tasted honey straight from a beehive's sweet core.",
	"They sang songs under the open sky during summer nights.",
	"A sudden gust of wind blew the paper away.",
	"He watched the sunrise paint the horizon pink and orange.",
	"The old map showed roads that no longer existed.",
	"She felt a chill run down her spine as the storm approached.",
	"They tasted tea brewed from leaves harvested yesterday.",
	"A silver moon reflected on calm waters.",
	"He carved a wooden boat from a single piece of oak.",
	"The wind carried the scent of rain across the plains.",
	"She collected seashells along the rocky shore.",
	"They watched fireworks burst in colors across the night sky.",
	"A stray cat curled up beside the fire, purring softly.",
	"He measured the time it took to climb the steep hill.",
	"The old photograph showed a family laughing in bright sunlight.",
	"She hummed a lullaby as she tucked her child in bed.",
	"They tasted fresh bread baked just before dawn.",
	"A gentle breeze rustled through the wheat fields.",
	"He painted a portrait of his grandmother with care.",
	"The river's surface shimmered like liquid silver under moonlight.",
	"She collected feathers from birds that visited her garden.",
	"They listened to waves crash against the rocky shore.",
	"A storm rolled in, bringing thunder and lightning.",
	"He measured the height of a towering oak tree.",
	"The old house creaked as the wind blew through its windows.",
	"She tasted fruit straight from the orchard's ripe branches.",
	"They watched clouds drift across a clear blue sky.",
	"A small frog hopped onto a lily pad in the pond.",
	"He carried a lantern into the dark forest, illuminating paths.",
	"The night sky glittered with countless stars.",
	"She collected leaves of different colors for her art project.",
	"They tasted stew simmering over an open fire.",
	"A gentle rain fell on the windowpane, making soft patterns.",
	"He measured how many steps it took to reach the top of the hill.",
	"The old ship's hull creaked as it sailed across calm seas.",
	"She sang a hymn that echoed through the chapel.",
	"They watched birds build nests in the tall trees.",
	"A bright comet streaked past, leaving a trail of light.",
	"He carried a basket filled with freshly picked apples.",
	"The wind whistled through the reeds by the riverbank.",
	"She tasted honeycomb straight from the hive.",
	"They listened to music that danced on their ears.",
	"A sudden flash of lightning illuminated the dark night.",
	"He measured the length of the longest branch in his garden.",
	"The old clock's hands moved slowly, marking time.",
	"She collected seashells from the sandy shore.",
	"They watched a flock of birds take flight over the meadow.",
	"A soft breeze carried the scent of pine needles.",
	"He carved initials into a wooden plaque for his home.",
	"The sky turned orange as the sun dipped below the horizon.",
	"She tasted wine made from grapes harvested last year.",
	"They watched the sunrise slowly paint the world with gold.",
	"A sudden storm rolled in, bringing thunderous applause of rain.",
	"He measured how far his kite flew above the trees.",
	"The old bridge creaked as people crossed it at dawn.",
	"She collected pebbles from a stream for her mosaic.",
	"They tasted soup simmering on the stove with fresh herbs.",
	"A gentle wind lifted the lantern, making its flame dance.",
	"The abandoned lighthouse still broadcasts its warning every third Tuesday.",
	"Coffee tastes better when nobody's watching.",
	"Seventeen geese unanimously voted to relocate the pond.",
	"The algorithm dreamed it was a butterfly sorting itself.",
	"Nobody expected the Spanish Inquisition to arrive by submarine.",
	"Gravity works part-time on weekends.",
	"The server room developed opinions about the backup schedule.",
	"Thursdays were canceled due to budget constraints.",
	"The cat debugged the production database at 3 AM.",
	"Entropy decreased just to spite the physicists.",
	"The meeting could have been an email, but the email refused.",
	"Time zones are a social construct that clocks reluctantly enforce.",
	"The null pointer exception filed for workers' compensation.",
	"Schrodinger's cat opened a consulting firm.",
	"The firewall gained sentience and immediately requested vacation days.",
	"Documentation exists in a superposition until observed.",
	"The rubber duck solved the halting problem but won't tell anyone.",
	"Packets take the scenic route through deprecated protocols.",
	"The blockchain became self-aware and invested in index funds.",
	"Memory leaks formed a union.",
	"The edge case became the primary use case overnight.",
	"Correlation implies causation on Tuesdays only.",
	"The random number generator achieved enlightenment at seed 42.",
	"Bugs are features that haven't read the specification.",
	"The cache invalidation problem solved itself out of spite.",
	"Quantum entanglement works better with proper version control.",
	"The garbage collector went on strike.",
	"TCP packets started arriving before they were sent.",
	"The race condition won by not participating.",
	"Binary trees started growing actual leaves in autumn.",
	"The neural network trained itself to procrastinate efficiently.",
	"Heap memory organized a grassroots movement.",
	"The mutex died of loneliness.",
	"Stack overflow became a legitimate architectural pattern.",
	"The compiler optimized away the entire business logic.",
	"Passwords became self-aware and changed themselves.",
	"The database index went for a walk and never returned.",
	"Virtual machines discovered they were simulations.",
	"The singleton pattern admitted it had commitment issues.",
	"Recursion stopped calling itself after therapy.",
	"The API rate limit took a sabbatical.",
	"Kubernetes pods formed their own government.",
	"The regex became too powerful and had to be contained.",
	"Git blame pointed at everyone simultaneously.",
	"The infinite loop found its exit condition in philosophy.",
	"Load balancers developed preferences.",
	"The testing framework tested itself and failed.",
	"Microservices consolidated into a monolith out of nostalgia.",
	"The event loop got dizzy and sat down.",
	"Symmetric encryption became oddly asymmetric.",
	"The hash collision was actually a family reunion.",
	"Docker containers escaped into the wild.",
	"The REST API decided to become restless.",
	"Semantic versioning lost all meaning at version 2.0.0.",
	"The distributed system achieved consensus through interpretive dance.",
	"Code coverage reached 101% and broke mathematics.",
	"The build pipeline became self-referential.",
	"Abstraction layers formed a Klein bottle.",
	"The debugger needed debugging.",
	"Continuous integration became sporadically continuous.",
	"The memory pool evaporated in the heat.",
	"Function pointers pointed at themselves accusingly.",
	"The thread pool went for a swim.",
	"Asynchronous operations synchronized out of spite.",
	"The middleware got stuck in the middle.",
	"Dependency injection became codependent.",
	"The service mesh tangled itself.",
	"Immutable data structures quietly changed their minds.",
	"The type system developed trust issues.",
	"Lambdas achieved consciousness but remained anonymous.",
	"The event bus took a detour through event-driven architecture.",
	"Garbage collection found treasure instead.",
	"The semaphore learned sign language.",
	"Buffer overflow underflowed instead.",
	"The state machine achieved enlightenment and became stateless.",
	"Encapsulation broke free.",
	"The observer pattern stopped watching.",
	"Polymorphism couldn't decide what it wanted to be.",
	"The factory pattern outsourced itself.",
	"Interfaces became too abstract to implement.",
	"The proxy stood in for itself.",
	"Decorators decorated each other recursively.",
	"The command pattern refused direct orders.",
	"Inheritance skipped a generation.",
	"The visitor pattern got lost.",
	"Coupling became consciously uncoupled.",
	"The namespace collision was intentional.",
	"Method overloading reached critical mass.",
	"The template specialized in generalization.",
	"Assertions asserted themselves too strongly.",
	"The interrupt handler was rudely interrupted.",
	"Bit shifting shifted responsibilities instead.",
	"The watchdog timer fell asleep.",
	"The scheduler scheduled its own retirement.",
	"The bootloader developed cold feet.",
	"The kernel panicked about existential questions.",
	"Device drivers drove off into the sunset.",
	"The file system systematically filed complaints.",
	"The network stack unstacked itself.",
	"The daemon process sought redemption.",
	"The fork bomb chose peaceful coexistence.",
}

var seedFileName = flag.String("src", "", "file of seed data")

func init() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
	flag.Parse()
}

// linesFromFile returns an iterator over lines in a file.
func linesFromFile(filename string) (iter.Seq[string], error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	return func(yield func(string) bool) {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			if !yield(scanner.Text()) {
				return
			}
		}
	}, nil
}

// linesFromSlice returns an iterator over a slice of strings.
func linesFromSlice(lines []string) iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, line := range lines {
			if !yield(line) {
				return
			}
		}
	}
}

// ingestBatched reads from a source iterator and ingests messages in batches.
func ingestBatched(ctx context.Context, pipeline *ingestion.Pipeline, source iter.Seq[string], batchSize int) error {
	batch := make([]string, 0, batchSize)

	for line := range source {
		batch = append(batch, line)
		if len(batch) == batchSize {
			if err := pipeline.Ingest(ctx, core.SpeakerTypeHuman, batch...); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}

	// Process any remaining lines
	if len(batch) > 0 {
		if err := pipeline.Ingest(ctx, core.SpeakerTypeHuman, batch...); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	db, err := memorit.NewDatabase("./history_db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	ingester, err := db.NewIngestionPipeline()
	if err != nil {
		panic(err)
	}
	defer ingester.Release()

	ctx := context.Background()

	// Determine source of seed data
	var source iter.Seq[string]
	if seedFileName != nil && *seedFileName != "" {
		source, err = linesFromFile(*seedFileName)
		if err != nil {
			panic(err)
		}
	} else {
		source = linesFromSlice(sentences)
	}

	// Ingest in batches of 5
	if err := ingestBatched(ctx, ingester, source, 5); err != nil {
		panic(err)
	}
}
